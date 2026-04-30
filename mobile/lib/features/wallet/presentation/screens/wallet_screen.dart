import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../invoicing/data/repositories/invoicing_repository_impl.dart';
import '../../../invoicing/presentation/providers/invoicing_providers.dart';
import '../../../invoicing/presentation/widgets/billing_profile_completion_modal.dart';
import '../../../invoicing/presentation/widgets/current_month_aggregate_card.dart';
import '../../data/exceptions/kyc_incomplete_exception.dart';
import '../../domain/entities/wallet_entity.dart';
import '../providers/wallet_provider.dart';
import '../widgets/wallet_commissions_section.dart';
import '../widgets/wallet_hero_card.dart';
import '../widgets/wallet_missions_section.dart';

/// Wallet screen — mirrors the web redesign: hero (total + stripe + payout),
/// missions section (3 cards + history), commissions section (3 cards +
/// history). Escrow rows are visually distinct with an amber left accent.
class WalletScreen extends ConsumerStatefulWidget {
  const WalletScreen({super.key});

  @override
  ConsumerState<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends ConsumerState<WalletScreen> {
  bool _payingOut = false;
  // Tracks the record id currently being retried (one at a time) so
  // the UI can show an inline spinner on the correct row. Holds a
  // payment-record id, NOT a proposal id.
  String? _retryingRecordId;

  Future<void> _requestPayout() async {
    // Proactive gate ORDER MATTERS — KYC first, billing second.
    // The backend enforces the same order so the user fixes their
    // actual blocker before round-tripping a doomed request.
    final asyncWallet = ref.read(walletProvider);
    final wallet = asyncWallet.valueOrNull;
    if (wallet != null && !wallet.payoutsEnabled) {
      if (!mounted) return;
      await _showKYCIncompleteDialog();
      return;
    }

    final completeness = ref.read(billingProfileCompletenessProvider);
    if (!completeness.isLoading && !completeness.isComplete) {
      if (!mounted) return;
      await showBillingProfileCompletionModal(
        context,
        missingFields: completeness.missingFields,
        message:
            'Complète ton profil de facturation pour pouvoir retirer.',
        returnTo: '/wallet',
      );
      return;
    }

    setState(() => _payingOut = true);
    try {
      final repo = ref.read(walletRepositoryProvider);
      await repo.requestPayout();
      ref.invalidate(walletProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(
              AppLocalizations.of(context)!.walletPayoutRequested,
            ),
          ),
        );
      }
    } on DioException catch (e) {
      // Defensive gates — the cached snapshots can be stale, so 403s
      // may still come back. Decode in the SAME ORDER as the backend.
      final kycIncomplete = tryDecodeKYCIncomplete(e);
      if (kycIncomplete != null) {
        ref.invalidate(walletProvider);
        if (mounted) {
          await _showKYCIncompleteDialog(message: kycIncomplete.message);
        }
        return;
      }
      final incomplete = tryDecodeBillingProfileIncomplete(e);
      if (incomplete != null) {
        ref.invalidate(billingProfileProvider);
        if (mounted) {
          await showBillingProfileCompletionModal(
            context,
            missingFields: incomplete.missingFields,
            message:
                'Complète ton profil de facturation pour pouvoir retirer.',
            returnTo: '/wallet',
          );
        }
      } else if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Payout failed: $e')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Payout failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _payingOut = false);
    }
  }

  /// Surfaces a small AlertDialog explaining the user must finish
  /// their Stripe onboarding before they can withdraw. Mirrors the
  /// BillingProfileCompletionModal pattern for consistency.
  Future<void> _showKYCIncompleteDialog({String? message}) async {
    await showDialog<void>(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: const Text(
            'Termine ton onboarding Stripe pour pouvoir retirer',
          ),
          content: Text(
            (message != null && message.isNotEmpty)
                ? message
                : "Avant de pouvoir retirer tes gains, finalise ton onboarding "
                    "Stripe sur la page Infos paiement. Les virements ne sont "
                    "activés qu'après vérification de ton identité par Stripe.",
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(),
              child: const Text('Plus tard'),
            ),
            ElevatedButton(
              onPressed: () {
                Navigator.of(dialogContext).pop();
                GoRouter.of(context).push(RoutePaths.paymentInfo);
              },
              child: const Text('Aller à Infos paiement'),
            ),
          ],
        );
      },
    );
  }

  Future<void> _retryTransfer(String recordId) async {
    setState(() => _retryingRecordId = recordId);
    try {
      final repo = ref.read(walletRepositoryProvider);
      await repo.retryFailedTransfer(recordId);
      ref.invalidate(walletProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Transfer retried')),
        );
      }
    } on DioException catch (e) {
      // 412 provider_kyc_incomplete is the most common real-world
      // failure mode (account exists but payouts_enabled=false).
      if (_isKYCIncomplete(e)) {
        if (mounted) {
          await _showKYCIncompleteDialog(
            message:
                'Termine ton onboarding Stripe pour pouvoir recevoir le virement.',
          );
        }
      } else if (mounted) {
        final code = _errorCode(e);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(_retryFailureCopy(code))),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Retry failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _retryingRecordId = null);
    }
  }

  /// Returns true when the 412 envelope carries the
  /// `provider_kyc_incomplete` discriminator. Distinct from the
  /// `kyc_incomplete` code returned by the payout flow.
  bool _isKYCIncomplete(DioException e) {
    if (e.response?.statusCode != 412) return false;
    return _errorCode(e) == 'provider_kyc_incomplete';
  }

  /// Reads `error` off the flat envelope produced by pkg/response.Error.
  String _errorCode(DioException e) {
    final body = e.response?.data;
    if (body is Map && body['error'] is String) {
      return body['error'] as String;
    }
    return '';
  }

  /// Maps the backend error code to the user-facing copy.
  String _retryFailureCopy(String code) {
    switch (code) {
      case 'transfer_not_retriable':
        return 'Ce transfert ne peut plus être relancé — la mission doit être terminée et le précédent transfert en échec.';
      case 'stripe_account_missing':
        return "Configure d'abord tes informations de paiement avant de relancer ce transfert.";
      case 'payment_record_not_found':
        return 'Ce transfert est introuvable.';
      case 'retry_failed':
        return 'Le virement a de nouveau échoué côté Stripe. Réessaie dans quelques minutes ou contacte le support.';
      default:
        return 'Erreur lors de la nouvelle tentative.';
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final asyncWallet = ref.watch(walletProvider);

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.walletTitle),
      ),
      body: asyncWallet.when(
        loading: () =>
            const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Error: $error'),
              const SizedBox(height: 8),
              ElevatedButton(
                onPressed: () => ref.invalidate(walletProvider),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        data: (wallet) => _buildContent(context, ref, wallet),
      ),
    );
  }

  Widget _buildContent(
    BuildContext context,
    WidgetRef ref,
    WalletOverview wallet,
  ) {
    final canWithdraw = ref.watch(
      hasPermissionProvider(OrgPermission.walletWithdraw),
    );
    final totalEarned =
        wallet.transferredAmount + wallet.commissions.paidCents;

    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const CurrentMonthAggregateCard(),
            const SizedBox(height: 16),
            WalletHeroCard(
              wallet: wallet,
              totalEarned: totalEarned,
              canWithdraw: canWithdraw,
              payingOut: _payingOut,
              onPayout: _requestPayout,
            ),
            const SizedBox(height: 24),
            WalletMissionsSection(
              wallet: wallet,
              retryingRecordId: _retryingRecordId,
              onRetry: _retryTransfer,
            ),
            if (!wallet.commissions.isEmpty ||
                wallet.commissionRecords.isNotEmpty) ...[
              const SizedBox(height: 24),
              WalletCommissionsSection(
                summary: wallet.commissions,
                records: wallet.commissionRecords,
              ),
            ],
          ],
        ),
      ),
    );
  }
}

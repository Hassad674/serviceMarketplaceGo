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
import '../../data/exceptions/commission_kyc_required_exception.dart';
import '../../data/exceptions/kyc_incomplete_exception.dart';
import '../../domain/entities/wallet_summary_entity.dart';
import '../providers/wallet_provider.dart';
import '../widgets/commission_kyc_required_dialog.dart';
import '../widgets/wallet_unified_header.dart';
import '../widgets/wallet_unified_history.dart';
import '../widgets/wallet_withdraw_result_sheet.dart';

/// Wallet screen — WALLET-UNIFY Run D refonte. Consumes the
/// consolidated GET /wallet/summary endpoint and wires the single
/// POST /wallet/withdraw flow through the existing KYC +
/// billing-profile gating modals (D1+D2). Mirrors the web Run C
/// experience: hero card + 3 stat cards + unified history. The
/// legacy per-row "Retirer" button on commission tiles is GONE —
/// the single hero CTA drains both legs.
class WalletScreen extends ConsumerStatefulWidget {
  const WalletScreen({super.key});

  @override
  ConsumerState<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends ConsumerState<WalletScreen> {
  bool _withdrawing = false;

  Future<void> _onWithdraw() async {
    if (_withdrawing) return;
    final l10n = AppLocalizations.of(context)!;
    final summary =
        ref.read(walletSummaryProvider(null)).valueOrNull;
    if (summary != null && summary.availableCents <= 0) return;

    // Billing-profile pre-flight. The web/wallet-unified-page flow
    // checks this first before firing the mutation so we don't
    // waste a Stripe round-trip.
    final completeness = ref.read(billingProfileCompletenessProvider);
    if (!completeness.isLoading && !completeness.isComplete) {
      if (!mounted) return;
      await showBillingProfileCompletionModal(
        context,
        missingFields: completeness.missingFields,
        message: l10n.walletUnifiedSubtitle,
        returnTo: '/wallet',
      );
      return;
    }

    setState(() => _withdrawing = true);
    try {
      final repo = ref.read(walletRepositoryProvider);
      final result = await repo.withdraw();
      ref.invalidate(walletSummaryProvider);
      ref.invalidate(walletProvider);
      if (!mounted) return;
      if (result.isPartialSuccess) {
        await showWalletWithdrawResultSheet(
          context: context,
          result: result,
        );
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.walletUnifiedToastPartial)),
        );
      } else if (result.isFullSuccess) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(l10n.walletUnifiedToastSuccess)),
        );
      }
    } on CommissionKYCRequiredException catch (kyc) {
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (dialogContext) => CommissionKYCRequiredDialog(
          onboardingUrl: kyc.onboardingUrl,
          onPaymentInfoTap: () => context.push(RoutePaths.paymentInfo),
        ),
      );
    } on DioException catch (e) {
      if (!mounted) return;
      final kycIncomplete = tryDecodeKYCIncomplete(e);
      if (kycIncomplete != null) {
        await _showKYCIncompleteDialog(message: kycIncomplete.message);
        return;
      }
      final billingIncomplete = tryDecodeBillingProfileIncomplete(e);
      if (billingIncomplete != null) {
        ref.invalidate(billingProfileProvider);
        if (!mounted) return;
        await showBillingProfileCompletionModal(
          context,
          missingFields: billingIncomplete.missingFields,
          message: l10n.walletUnifiedSubtitle,
          returnTo: '/wallet',
        );
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Withdraw failed: ${e.message}')),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Withdraw failed: $e')),
      );
    } finally {
      if (mounted) setState(() => _withdrawing = false);
    }
  }

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

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final asyncSummary = ref.watch(walletSummaryProvider(null));

    return Scaffold(
      appBar: AppBar(
        leading: const IconButton(
          icon: Icon(Icons.menu),
          onPressed: openShellDrawer,
        ),
        title: Text(l10n.walletUnifiedTitle),
      ),
      body: asyncSummary.when(
        loading: () =>
            const Center(child: CircularProgressIndicator()),
        error: (error, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Error: $error'),
              const SizedBox(height: 8),
              ElevatedButton(
                onPressed: () =>
                    ref.invalidate(walletSummaryProvider),
                child: Text(l10n.retry),
              ),
            ],
          ),
        ),
        data: (summary) => _buildContent(summary),
      ),
    );
  }

  Widget _buildContent(WalletSummary summary) {
    final canWithdraw = ref.watch(
      hasPermissionProvider(OrgPermission.walletWithdraw),
    );
    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            WalletUnifiedHeader(
              summary: summary,
              payoutPending: _withdrawing,
              canWithdraw: canWithdraw,
              onWithdraw: _onWithdraw,
            ),
            const SizedBox(height: 16),
            const WalletUnifiedHistory(),
          ],
        ),
      ),
    );
  }
}

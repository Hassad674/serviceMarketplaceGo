import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/utils/permissions.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/wallet_entity.dart';
import '../providers/wallet_provider.dart';

// ---------------------------------------------------------------------------
// Wallet screen — mirrors the web redesign: hero (total + stripe + payout),
// missions section (3 cards + history), commissions section (3 cards +
// history). Escrow rows are visually distinct with an amber left accent.
// ---------------------------------------------------------------------------

class WalletScreen extends ConsumerStatefulWidget {
  const WalletScreen({super.key});

  @override
  ConsumerState<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends ConsumerState<WalletScreen> {
  bool _payingOut = false;
  String? _retryingProposalId;

  Future<void> _requestPayout() async {
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

  Future<void> _retryTransfer(String proposalId) async {
    setState(() => _retryingProposalId = proposalId);
    try {
      final repo = ref.read(walletRepositoryProvider);
      await repo.retryFailedTransfer(proposalId);
      ref.invalidate(walletProvider);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Transfer retried')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Retry failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _retryingProposalId = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final asyncWallet = ref.watch(walletProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.menu),
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
        data: (wallet) => _buildContent(context, ref, l10n, wallet),
      ),
    );
  }

  Widget _buildContent(
    BuildContext context,
    WidgetRef ref,
    AppLocalizations l10n,
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
            // Hero — total earned + stripe status + payout CTA
            _WalletHero(
              wallet: wallet,
              totalEarned: totalEarned,
              canWithdraw: canWithdraw,
              payingOut: _payingOut,
              onPayout: _requestPayout,
            ),
            const SizedBox(height: 24),

            // Missions section — 3 cards + history
            _MissionsSection(
              wallet: wallet,
              retryingProposalId: _retryingProposalId,
              onRetry: _retryTransfer,
            ),

            // Commissions section — hidden when zero activity
            if (!wallet.commissions.isEmpty ||
                wallet.commissionRecords.isNotEmpty) ...[
              const SizedBox(height: 24),
              _CommissionSection(
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

// ---------------------------------------------------------------------------
// Hero — title, total earned, compact stripe status, payout CTA
// ---------------------------------------------------------------------------

class _WalletHero extends StatelessWidget {
  const _WalletHero({
    required this.wallet,
    required this.totalEarned,
    required this.canWithdraw,
    required this.payingOut,
    required this.onPayout,
  });

  final WalletOverview wallet;
  final int totalEarned;
  final bool canWithdraw;
  final bool payingOut;
  final VoidCallback onPayout;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasAccount = wallet.stripeAccountId.isNotEmpty;
    final canClick = canWithdraw && wallet.availableAmount > 0;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Title row
          Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: const Color(0xFFF43F5E).withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
                child: const Icon(
                  Icons.account_balance_wallet_outlined,
                  color: Color(0xFFF43F5E),
                  size: 20,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      AppLocalizations.of(context)!.walletTitle,
                      style: theme.textTheme.titleLarge?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    Text(
                      'Your mission and referral earnings',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurface
                            .withValues(alpha: 0.6),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 20),

          // Total earned
          Text(
            'TOTAL EARNINGS',
            style: theme.textTheme.labelSmall?.copyWith(
              letterSpacing: 0.8,
              fontWeight: FontWeight.w600,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
            ),
          ),
          const SizedBox(height: 4),
          Text(
            WalletOverview.formatCents(totalEarned),
            style: theme.textTheme.displaySmall?.copyWith(
              fontWeight: FontWeight.w800,
              fontFamily: 'monospace',
            ),
          ),
          const SizedBox(height: 20),

          // Stripe status line
          _StripeStatusLine(
            hasAccount: hasAccount,
            payoutsEnabled: wallet.payoutsEnabled,
          ),
          const SizedBox(height: 16),

          // Payout CTA
          SizedBox(
            width: double.infinity,
            height: 48,
            child: ElevatedButton.icon(
              onPressed: (payingOut || !canClick) ? null : onPayout,
              icon: payingOut
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor:
                            AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : const Icon(Icons.arrow_downward, size: 20),
              label: Text(
                '${AppLocalizations.of(context)!.walletRequestPayout} '
                '${WalletOverview.formatCents(wallet.availableAmount)}',
                style: const TextStyle(
                  fontWeight: FontWeight.w600,
                  fontSize: 15,
                ),
              ),
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFFF43F5E),
                foregroundColor: Colors.white,
                disabledBackgroundColor: const Color(0xFFF43F5E)
                    .withValues(alpha: 0.4),
                disabledForegroundColor: Colors.white,
                shape: RoundedRectangleBorder(
                  borderRadius:
                      BorderRadius.circular(AppTheme.radiusLg),
                ),
              ),
            ),
          ),
          if (wallet.availableAmount == 0)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                'No funds available to withdraw',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurface
                      .withValues(alpha: 0.5),
                ),
              ),
            ),
          if (wallet.availableAmount > 0 && !canWithdraw)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                AppLocalizations.of(context)!.permissionDeniedWithdraw,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _StripeStatusLine extends StatelessWidget {
  const _StripeStatusLine({
    required this.hasAccount,
    required this.payoutsEnabled,
  });

  final bool hasAccount;
  final bool payoutsEnabled;

  @override
  Widget build(BuildContext context) {
    final IconData icon;
    final Color color;
    final String label;
    if (hasAccount && payoutsEnabled) {
      icon = Icons.check_circle;
      color = const Color(0xFF22C55E);
      label = 'Stripe account ready — payouts enabled';
    } else if (hasAccount) {
      icon = Icons.warning_amber_rounded;
      color = const Color(0xFFF59E0B);
      label = 'Stripe account verifying';
    } else {
      icon = Icons.cancel;
      color = const Color(0xFFEF4444);
      label = 'Stripe account not configured';
    }

    return Row(
      children: [
        Icon(icon, size: 16, color: color),
        const SizedBox(width: 6),
        Flexible(
          child: Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.7),
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Missions section — 3 balance cards + history
// ---------------------------------------------------------------------------

class _MissionsSection extends StatelessWidget {
  const _MissionsSection({
    required this.wallet,
    required this.retryingProposalId,
    required this.onRetry,
  });

  final WalletOverview wallet;
  final String? retryingProposalId;
  final Future<void> Function(String proposalId) onRetry;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const _SectionHeader(
          icon: Icons.work_outline,
          title: 'My missions',
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: _BalanceCard(
                icon: Icons.lock_outline,
                label: 'Escrow',
                amount: wallet.escrowAmount,
                color: const Color(0xFFF59E0B),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _BalanceCard(
                icon: Icons.account_balance_wallet_outlined,
                label: 'Available',
                amount: wallet.availableAmount,
                color: const Color(0xFF22C55E),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _BalanceCard(
                icon: Icons.send_outlined,
                label: 'Transferred',
                amount: wallet.transferredAmount,
                color: const Color(0xFF2563EB),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        _HistoryCard(
          title: 'Mission history',
          subtitle: 'All your missions — from escrow to transfer',
          emptyLabel: 'No missions yet',
          isEmpty: wallet.records.isEmpty,
          children: [
            for (final r in wallet.records)
              _MissionTile(
                record: r,
                retrying: retryingProposalId == r.proposalId,
                onRetry: () => onRetry(r.proposalId),
              ),
          ],
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Commission section — mirror of missions for apporteur earnings
// ---------------------------------------------------------------------------

class _CommissionSection extends StatelessWidget {
  const _CommissionSection({required this.summary, required this.records});

  final CommissionWallet summary;
  final List<CommissionRecord> records;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const _SectionHeader(
          icon: Icons.auto_awesome,
          title: 'My referral commissions',
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: _BalanceCard(
                icon: Icons.schedule,
                label: 'Pending',
                amount:
                    summary.pendingCents + summary.pendingKycCents,
                color: const Color(0xFFF59E0B),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _BalanceCard(
                icon: Icons.verified_outlined,
                label: 'Received',
                amount: summary.paidCents,
                color: const Color(0xFF22C55E),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: _BalanceCard(
                icon: Icons.undo,
                label: 'Clawed back',
                amount: summary.clawedBackCents,
                color: const Color(0xFF2563EB),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        _HistoryCard(
          title: 'Commission history',
          subtitle: 'Every referral you have facilitated',
          emptyLabel: 'No commissions yet',
          isEmpty: records.isEmpty,
          children: [
            for (final r in records) _CommissionTile(record: r),
          ],
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Reusable primitives
// ---------------------------------------------------------------------------

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.icon, required this.title});

  final IconData icon;
  final String title;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Icon(icon, size: 18, color: const Color(0xFFF43F5E)),
        const SizedBox(width: 8),
        Text(
          title,
          style: Theme.of(context).textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
      ],
    );
  }
}

class _BalanceCard extends StatelessWidget {
  const _BalanceCard({
    required this.icon,
    required this.label,
    required this.amount,
    required this.color,
  });

  final IconData icon;
  final String label;
  final int amount;
  final Color color;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 32,
            height: 32,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(AppTheme.radiusSm),
            ),
            child: Icon(icon, color: color, size: 16),
          ),
          const SizedBox(height: 8),
          Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color:
                  theme.colorScheme.onSurface.withValues(alpha: 0.6),
              fontWeight: FontWeight.w500,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 2),
          Text(
            WalletOverview.formatCents(amount),
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.bold,
              fontFamily: 'monospace',
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ],
      ),
    );
  }
}

class _HistoryCard extends StatelessWidget {
  const _HistoryCard({
    required this.title,
    required this.subtitle,
    required this.emptyLabel,
    required this.isEmpty,
    required this.children,
  });

  final String title;
  final String subtitle;
  final String emptyLabel;
  final bool isEmpty;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  subtitle,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface
                        .withValues(alpha: 0.5),
                  ),
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          if (isEmpty)
            Padding(
              padding: const EdgeInsets.all(24),
              child: Center(
                child: Text(
                  emptyLabel,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurface
                        .withValues(alpha: 0.5),
                  ),
                ),
              ),
            )
          else
            ...children,
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Mission tile — with amber left accent when in escrow
// ---------------------------------------------------------------------------

class _MissionTile extends StatelessWidget {
  const _MissionTile({
    required this.record,
    required this.retrying,
    required this.onRetry,
  });

  final WalletRecord record;
  final bool retrying;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isFailed = record.transferStatus == 'failed';
    final isCompleted = record.transferStatus == 'completed';
    final isInEscrow = !isFailed && !isCompleted;

    final Color accentColor = isFailed
        ? const Color(0xFFEF4444)
        : isInEscrow
            ? const Color(0xFFF59E0B)
            : Colors.transparent;

    final title = record.proposalTitle.isNotEmpty
        ? record.proposalTitle
        : 'Mission ${_formatDate(record.createdAt)}';

    return Container(
      decoration: BoxDecoration(
        border: Border(
          left: BorderSide(color: accentColor, width: 4),
          bottom: BorderSide(
            color: theme.dividerColor.withValues(alpha: 0.3),
          ),
        ),
      ),
      padding: const EdgeInsets.fromLTRB(12, 12, 16, 12),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(fontWeight: FontWeight.w600),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                if (isInEscrow)
                  Text(
                    'In escrow — mission in progress',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: const Color(0xFFB45309),
                      fontWeight: FontWeight.w500,
                    ),
                  )
                else if (isFailed)
                  Text(
                    'Transfer failed',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: const Color(0xFFDC2626),
                      fontWeight: FontWeight.w600,
                    ),
                  )
                else
                  Text(
                    'Transferred',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: const Color(0xFF15803D),
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            WalletOverview.formatCents(record.netAmount),
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w700,
              fontFamily: 'monospace',
            ),
          ),
          if (isFailed) ...[
            const SizedBox(width: 4),
            IconButton(
              tooltip: 'Retry transfer',
              onPressed: retrying ? null : onRetry,
              icon: retrying
                  ? const SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(
                      Icons.refresh,
                      size: 20,
                      color: Color(0xFFDC2626),
                    ),
              visualDensity: VisualDensity.compact,
            ),
          ],
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Commission tile — with amber left accent when pending, date-based label
// ---------------------------------------------------------------------------

class _CommissionTile extends StatelessWidget {
  const _CommissionTile({required this.record});

  final CommissionRecord record;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final chip = _commissionChip(record.status);

    final bool isPending =
        record.status == 'pending' || record.status == 'pending_kyc';
    final bool isClawed = record.status == 'clawed_back';

    final Color accentColor = isPending
        ? const Color(0xFFF59E0B)
        : isClawed
            ? const Color(0xFF2563EB)
            : Colors.transparent;

    return Container(
      decoration: BoxDecoration(
        border: Border(
          left: BorderSide(color: accentColor, width: 4),
          bottom: BorderSide(
            color: theme.dividerColor.withValues(alpha: 0.3),
          ),
        ),
      ),
      padding: const EdgeInsets.fromLTRB(12, 12, 16, 12),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Commission ${_formatDate(record.createdAt)}',
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(fontWeight: FontWeight.w600),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  'on ${WalletOverview.formatCents(record.grossAmountCents)} of mission',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface
                        .withValues(alpha: 0.5),
                  ),
                ),
              ],
            ),
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                WalletOverview.formatCents(record.commissionCents),
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: FontWeight.w700,
                  fontFamily: 'monospace',
                ),
              ),
              const SizedBox(height: 4),
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 8,
                  vertical: 2,
                ),
                decoration: BoxDecoration(
                  color: chip.color.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  chip.label,
                  style: TextStyle(
                    fontSize: 10,
                    color: chip.color,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
            ],
          ),
          if (record.referralId.isNotEmpty) ...[
            const SizedBox(width: 4),
            Icon(
              Icons.chevron_right,
              size: 20,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.4),
            ),
          ],
        ],
      ),
    );
  }

  ({String label, Color color}) _commissionChip(String status) {
    switch (status) {
      case 'paid':
        return (label: 'Received', color: const Color(0xFF10B981));
      case 'pending':
        return (label: 'Pending', color: const Color(0xFFF59E0B));
      case 'pending_kyc':
        return (label: 'KYC required', color: const Color(0xFFEA580C));
      case 'clawed_back':
        return (label: 'Clawed back', color: const Color(0xFF3B82F6));
      case 'failed':
        return (label: 'Failed', color: const Color(0xFFEF4444));
      case 'cancelled':
        return (label: 'Cancelled', color: const Color(0xFF64748B));
      default:
        return (label: status, color: const Color(0xFF64748B));
    }
  }
}

String _formatDate(DateTime d) {
  final dd = d.day.toString().padLeft(2, '0');
  final mm = d.month.toString().padLeft(2, '0');
  final yy = d.year.toString();
  return '$dd/$mm/$yy';
}

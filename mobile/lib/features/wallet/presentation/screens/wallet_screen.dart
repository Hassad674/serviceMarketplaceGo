import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/wallet_entity.dart';
import '../providers/wallet_provider.dart';

// ---------------------------------------------------------------------------
// Wallet screen
// ---------------------------------------------------------------------------

/// Wallet page showing Stripe account status, balance cards,
/// payout button, and transaction history.
class WalletScreen extends ConsumerStatefulWidget {
  const WalletScreen({super.key});

  @override
  ConsumerState<WalletScreen> createState() => _WalletScreenState();
}

class _WalletScreenState extends ConsumerState<WalletScreen> {
  bool _payingOut = false;

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
        data: (wallet) => _buildContent(context, l10n, wallet),
      ),
    );
  }

  Widget _buildContent(
    BuildContext context,
    AppLocalizations l10n,
    WalletOverview wallet,
  ) {
    return SafeArea(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Stripe account status
            _AccountStatusRow(wallet: wallet, l10n: l10n),
            const SizedBox(height: 16),

            // Balance cards
            _BalanceCards(wallet: wallet, l10n: l10n),
            const SizedBox(height: 16),

            // Payout button
            if (wallet.payoutsEnabled && wallet.availableAmount > 0)
              _PayoutButton(
                amount: wallet.availableAmount,
                loading: _payingOut,
                onPressed: _requestPayout,
              ),
            const SizedBox(height: 24),

            // Transaction history
            _TransactionHistory(
              records: wallet.records,
              l10n: l10n,
            ),
          ],
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Account status row
// ---------------------------------------------------------------------------

class _AccountStatusRow extends StatelessWidget {
  const _AccountStatusRow({
    required this.wallet,
    required this.l10n,
  });

  final WalletOverview wallet;
  final AppLocalizations l10n;

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
      child: Row(
        children: [
          Icon(
            Icons.account_balance_outlined,
            size: 20,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
          ),
          const SizedBox(width: 8),
          Text(
            l10n.walletStripeAccount,
            style: theme.textTheme.bodyMedium
                ?.copyWith(fontWeight: FontWeight.w500),
          ),
          const Spacer(),
          _StatusChip(
            enabled: wallet.chargesEnabled,
            label: l10n.walletCharges,
          ),
          const SizedBox(width: 6),
          _StatusChip(
            enabled: wallet.payoutsEnabled,
            label: l10n.walletPayouts,
          ),
        ],
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({
    required this.enabled,
    required this.label,
  });

  final bool enabled;
  final String label;

  @override
  Widget build(BuildContext context) {
    final bg =
        enabled ? const Color(0xFFECFDF5) : const Color(0xFFFEF2F2);
    final fg =
        enabled ? const Color(0xFF15803D) : const Color(0xFFDC2626);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: fg,
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Balance cards
// ---------------------------------------------------------------------------

class _BalanceCards extends StatelessWidget {
  const _BalanceCards({
    required this.wallet,
    required this.l10n,
  });

  final WalletOverview wallet;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Expanded(
          child: _BalanceCard(
            label: l10n.walletEscrow,
            amount: wallet.escrowAmount,
            color: const Color(0xFFF59E0B),
            icon: Icons.lock_outline,
          ),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: _BalanceCard(
            label: l10n.walletAvailable,
            amount: wallet.availableAmount,
            color: const Color(0xFF22C55E),
            icon: Icons.account_balance_wallet_outlined,
          ),
        ),
        const SizedBox(width: 8),
        Expanded(
          child: _BalanceCard(
            label: l10n.walletTransferred,
            amount: wallet.transferredAmount,
            color: const Color(0xFF2563EB),
            icon: Icons.send_outlined,
          ),
        ),
      ],
    );
  }
}

class _BalanceCard extends StatelessWidget {
  const _BalanceCard({
    required this.label,
    required this.amount,
    required this.color,
    required this.icon,
  });

  final String label;
  final int amount;
  final Color color;
  final IconData icon;

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
        children: [
          Container(
            width: 36,
            height: 36,
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
            child: Icon(icon, color: color, size: 18),
          ),
          const SizedBox(height: 8),
          Text(
            WalletOverview.formatCents(amount),
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.bold,
              fontFamily: 'monospace',
            ),
            textAlign: TextAlign.center,
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color:
                  theme.colorScheme.onSurface.withValues(alpha: 0.5),
            ),
            textAlign: TextAlign.center,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Payout button
// ---------------------------------------------------------------------------

class _PayoutButton extends StatelessWidget {
  const _PayoutButton({
    required this.amount,
    required this.loading,
    required this.onPressed,
  });

  final int amount;
  final bool loading;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return SizedBox(
      width: double.infinity,
      height: 48,
      child: ElevatedButton.icon(
        onPressed: loading ? null : onPressed,
        icon: loading
            ? const SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(
                  strokeWidth: 2,
                  valueColor:
                      AlwaysStoppedAnimation<Color>(Colors.white),
                ),
              )
            : const Icon(Icons.arrow_upward, size: 20),
        label: Text(
          '${l10n.walletRequestPayout} '
          '${WalletOverview.formatCents(amount)}',
          style: const TextStyle(
            fontWeight: FontWeight.w600,
            fontSize: 15,
          ),
        ),
        style: ElevatedButton.styleFrom(
          backgroundColor: const Color(0xFFF43F5E),
          foregroundColor: Colors.white,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Transaction history
// ---------------------------------------------------------------------------

class _TransactionHistory extends StatelessWidget {
  const _TransactionHistory({
    required this.records,
    required this.l10n,
  });

  final List<WalletRecord> records;
  final AppLocalizations l10n;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.walletTransactionHistory,
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 12),
        if (records.isEmpty)
          Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Text(
                l10n.walletNoTransactions,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: theme.colorScheme.onSurface
                      .withValues(alpha: 0.5),
                ),
              ),
            ),
          )
        else
          for (final r in records) _TransactionTile(record: r),
      ],
    );
  }
}

class _TransactionTile extends StatelessWidget {
  const _TransactionTile({required this.record});

  final WalletRecord record;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final title =
        record.proposalTitle.isNotEmpty
            ? record.proposalTitle
            : record.proposalId.substring(
                0,
                record.proposalId.length > 8
                    ? 8
                    : record.proposalId.length,
              );

    return Container(
      margin: const EdgeInsets.only(bottom: 8),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: theme.dividerColor.withValues(alpha: 0.5),
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: theme.textTheme.bodyMedium
                      ?.copyWith(fontWeight: FontWeight.w500),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                const SizedBox(height: 2),
                Text(
                  record.transferStatus,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurface
                        .withValues(alpha: 0.5),
                  ),
                ),
              ],
            ),
          ),
          Text(
            WalletOverview.formatCents(record.netAmount),
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w600,
              fontFamily: 'monospace',
            ),
          ),
        ],
      ),
    );
  }
}

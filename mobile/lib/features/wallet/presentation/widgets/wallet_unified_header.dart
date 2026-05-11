import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/wallet_summary_entity.dart';

/// WalletUnifiedHeader — purely presentational header for the
/// WALLET-UNIFY Run D refonte on mobile. Mirrors the web
/// `WalletUnifiedHeader` (Run C): a consolidated "Portefeuille"
/// hero card (icon + title + subtitle + total + single Retirer
/// button) plus a 3-up row of stat cards below it.
///
/// The host screen (`WalletScreen`) owns ALL state — mutation
/// pending, billing-profile modal, KYC modal, partial-success
/// sheet. This widget stays renderable in tests without a
/// QueryClient / Riverpod scope.
class WalletUnifiedHeader extends StatelessWidget {
  const WalletUnifiedHeader({
    super.key,
    required this.summary,
    required this.payoutPending,
    required this.canWithdraw,
    required this.onWithdraw,
  });

  final WalletSummary summary;
  final bool payoutPending;
  final bool canWithdraw;
  final VoidCallback onWithdraw;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _HeroCard(
          totalCents: summary.totalCents,
          availableCents: summary.availableCents,
          payoutPending: payoutPending,
          canWithdraw: canWithdraw,
          onWithdraw: onWithdraw,
        ),
        const SizedBox(height: 12),
        _StatCardsRow(
          escrowedCents: summary.escrowedCents,
          availableCents: summary.availableCents,
          transmittedCents: summary.transmittedCents,
        ),
      ],
    );
  }
}

class _HeroCard extends StatelessWidget {
  const _HeroCard({
    required this.totalCents,
    required this.availableCents,
    required this.payoutPending,
    required this.canWithdraw,
    required this.onWithdraw,
  });

  final int totalCents;
  final int availableCents;
  final bool payoutPending;
  final bool canWithdraw;
  final VoidCallback onWithdraw;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      key: const ValueKey('wallet-unified-hero'),
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
          _TitleRow(),
          const SizedBox(height: 16),
          _TotalAmount(totalCents: totalCents),
          const SizedBox(height: 16),
          _WithdrawButton(
            availableCents: availableCents,
            canWithdraw: canWithdraw,
            payoutPending: payoutPending,
            onWithdraw: onWithdraw,
          ),
        ],
      ),
    );
  }
}

class _TitleRow extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Row(
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: theme.colorScheme.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          child: Icon(
            Icons.account_balance_wallet_outlined,
            color: theme.colorScheme.primary,
            size: 20,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                l10n.walletUnifiedTitle,
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.bold,
                  color: theme.colorScheme.onSurface,
                ),
              ),
              Text(
                l10n.walletUnifiedSubtitle,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _TotalAmount extends StatelessWidget {
  const _TotalAmount({required this.totalCents});

  final int totalCents;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.walletUnifiedTotalEarned.toUpperCase(),
          style: theme.textTheme.bodySmall?.copyWith(
            fontSize: 11,
            letterSpacing: 0.8,
            fontWeight: FontWeight.w600,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
          ),
        ),
        const SizedBox(height: 4),
        Text(
          formatWalletSummaryCents(totalCents),
          key: const ValueKey('wallet-unified-total'),
          style: theme.textTheme.displaySmall?.copyWith(
            fontWeight: FontWeight.w800,
            fontFamily: 'monospace',
            color: theme.colorScheme.onSurface,
          ),
        ),
      ],
    );
  }
}

class _WithdrawButton extends StatelessWidget {
  const _WithdrawButton({
    required this.availableCents,
    required this.canWithdraw,
    required this.payoutPending,
    required this.onWithdraw,
  });

  final int availableCents;
  final bool canWithdraw;
  final bool payoutPending;
  final VoidCallback onWithdraw;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final canClick = canWithdraw && availableCents > 0 && !payoutPending;
    final label = payoutPending
        ? l10n.walletUnifiedWithdrawing
        : '${l10n.walletUnifiedWithdraw} ${formatWalletSummaryCents(availableCents)}';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        SizedBox(
          height: 48,
          child: ElevatedButton.icon(
            key: const ValueKey('wallet-unified-withdraw'),
            onPressed: canClick ? onWithdraw : null,
            icon: payoutPending
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
              label,
              style: const TextStyle(
                fontWeight: FontWeight.w600,
                fontSize: 15,
              ),
            ),
            style: ElevatedButton.styleFrom(
              backgroundColor: theme.colorScheme.primary,
              foregroundColor: Colors.white,
              disabledBackgroundColor:
                  theme.colorScheme.primary.withValues(alpha: 0.4),
              disabledForegroundColor: Colors.white,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTheme.radiusLg),
              ),
            ),
          ),
        ),
        if (availableCents == 0)
          Padding(
            padding: const EdgeInsets.only(top: 6),
            child: Text(
              l10n.walletUnifiedNoFunds,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
              ),
            ),
          ),
        if (availableCents > 0 && !canWithdraw)
          Padding(
            padding: const EdgeInsets.only(top: 6),
            child: Text(
              l10n.permissionDeniedWithdraw,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.error,
              ),
            ),
          ),
      ],
    );
  }
}

class _StatCardsRow extends StatelessWidget {
  const _StatCardsRow({
    required this.escrowedCents,
    required this.availableCents,
    required this.transmittedCents,
  });

  final int escrowedCents;
  final int availableCents;
  final int transmittedCents;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final accent = theme.extension<AppColors>();
    return IntrinsicHeight(
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Expanded(
            child: _StatCard(
              label: l10n.walletUnifiedCardEscrowed,
              hint: l10n.walletUnifiedCardEscrowedHint,
              cents: escrowedCents,
              toneBg:
                  theme.colorScheme.onSurface.withValues(alpha: 0.06),
              toneFg:
                  theme.colorScheme.onSurface.withValues(alpha: 0.7),
              testKey: const ValueKey('wallet-stat-escrowed'),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _StatCard(
              label: l10n.walletUnifiedCardAvailable,
              hint: l10n.walletUnifiedCardAvailableHint,
              cents: availableCents,
              toneBg: (accent?.success ?? theme.colorScheme.primary)
                  .withValues(alpha: 0.12),
              toneFg: accent?.success ?? theme.colorScheme.primary,
              testKey: const ValueKey('wallet-stat-available'),
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _StatCard(
              label: l10n.walletUnifiedCardTransmitted,
              hint: l10n.walletUnifiedCardTransmittedHint,
              cents: transmittedCents,
              toneBg:
                  theme.colorScheme.primary.withValues(alpha: 0.12),
              toneFg: theme.colorScheme.primary,
              testKey: const ValueKey('wallet-stat-transmitted'),
            ),
          ),
        ],
      ),
    );
  }
}

class _StatCard extends StatelessWidget {
  const _StatCard({
    required this.label,
    required this.hint,
    required this.cents,
    required this.toneBg,
    required this.toneFg,
    required this.testKey,
  });

  final String label;
  final String hint;
  final int cents;
  final Color toneBg;
  final Color toneFg;
  final ValueKey testKey;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      key: testKey,
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
            padding:
                const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              color: toneBg,
              borderRadius:
                  BorderRadius.circular(AppTheme.radiusSm),
            ),
            child: Text(
              label.toUpperCase(),
              style: theme.textTheme.labelSmall?.copyWith(
                color: toneFg,
                fontSize: 10,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.5,
              ),
            ),
          ),
          const SizedBox(height: 8),
          Text(
            formatWalletSummaryCents(cents),
            style: theme.textTheme.titleLarge?.copyWith(
              fontWeight: FontWeight.w700,
              fontFamily: 'monospace',
              color: theme.colorScheme.onSurface,
            ),
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
          const SizedBox(height: 2),
          Text(
            hint,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurface.withValues(alpha: 0.55),
              fontSize: 11,
            ),
            maxLines: 2,
          ),
        ],
      ),
    );
  }
}

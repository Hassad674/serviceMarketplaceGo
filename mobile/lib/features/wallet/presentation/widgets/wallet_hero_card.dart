import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/router/app_router.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/wallet_entity.dart';
import '../../../../core/theme/app_palette.dart';

/// Hero card: title row, total earnings, Stripe status line, payout
/// CTA, and quick links to billing profile + payment info screens.
///
/// CRITICAL: this widget preserves the KYC payout gate contract — the
/// payout button is disabled when [canWithdraw] is false or the
/// available balance is zero, and the parent's [onPayout] handler is
/// responsible for the proactive KYC + billing-profile gates before
/// hitting the wallet endpoint.
class WalletHeroCard extends StatelessWidget {
  const WalletHeroCard({
    super.key,
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
          _buildTitleRow(context),
          const SizedBox(height: 20),
          _buildTotalEarned(context),
          const SizedBox(height: 20),
          WalletStripeStatusLine(
            hasAccount: hasAccount,
            payoutsEnabled: wallet.payoutsEnabled,
          ),
          const SizedBox(height: 16),
          _buildPayoutCta(context, canClick),
          if (wallet.availableAmount == 0)
            _emptyBalanceLabel(context),
          if (wallet.availableAmount > 0 && !canWithdraw)
            _permissionDeniedLabel(context),
          const SizedBox(height: 16),
          Divider(
            height: 1,
            color: theme.dividerColor.withValues(alpha: 0.5),
          ),
          const SizedBox(height: 12),
          _buildQuickLinks(context),
        ],
      ),
    );
  }

  Widget _buildTitleRow(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: AppPalette.rose500.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
          child: const Icon(
            Icons.account_balance_wallet_outlined,
            color: AppPalette.rose500,
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
    );
  }

  Widget _buildTotalEarned(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'TOTAL EARNINGS',
          // labelSmall is not overridden in the Soleil textTheme so the
          // base style can fall back to a Material default that renders
          // white on light bg. Anchor on bodySmall (which IS defined and
          // points to the muted foreground) to guarantee a visible tone.
          style: theme.textTheme.bodySmall?.copyWith(
            fontSize: 11,
            letterSpacing: 0.8,
            fontWeight: FontWeight.w600,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
          ),
        ),
        const SizedBox(height: 4),
        Text(
          WalletOverview.formatCents(totalEarned),
          // displaySmall is not overridden in the Soleil textTheme — it
          // falls back to the Material default whose color can be white
          // on the light ivoire surface. Force onSurface (encre) so the
          // hero balance number is always legible.
          style: theme.textTheme.displaySmall?.copyWith(
            fontWeight: FontWeight.w800,
            fontFamily: 'monospace',
            color: theme.colorScheme.onSurface,
          ),
        ),
      ],
    );
  }

  Widget _buildPayoutCta(BuildContext context, bool canClick) {
    return SizedBox(
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
          backgroundColor: AppPalette.rose500,
          foregroundColor: Colors.white,
          disabledBackgroundColor:
              AppPalette.rose500.withValues(alpha: 0.4),
          disabledForegroundColor: Colors.white,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          ),
        ),
      ),
    );
  }

  Widget _emptyBalanceLabel(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Text(
        'No funds available to withdraw',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
        ),
      ),
    );
  }

  Widget _permissionDeniedLabel(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(top: 8),
      child: Text(
        AppLocalizations.of(context)!.permissionDeniedWithdraw,
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.error,
        ),
      ),
    );
  }

  Widget _buildQuickLinks(BuildContext context) {
    final theme = Theme.of(context);
    return Wrap(
      spacing: 16,
      runSpacing: 8,
      children: [
        InkWell(
          onTap: () => context.push(
            '${RoutePaths.billingProfile}?return_to=/wallet',
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.edit_outlined,
                size: 14,
                color: theme.colorScheme.onSurface
                    .withValues(alpha: 0.7),
              ),
              const SizedBox(width: 6),
              Text(
                'Modifier mes infos de facturation',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurface
                      .withValues(alpha: 0.75),
                ),
              ),
            ],
          ),
        ),
        InkWell(
          onTap: () => context.push(RoutePaths.paymentInfo),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.shield_outlined,
                size: 14,
                color: theme.colorScheme.onSurface
                    .withValues(alpha: 0.7),
              ),
              const SizedBox(width: 6),
              Text(
                'Infos de paiement / KYC',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurface
                      .withValues(alpha: 0.75),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

/// Single line summarising the Stripe Connect account state. Three
/// states: ready (green check), verifying (amber), not configured
/// (red).
class WalletStripeStatusLine extends StatelessWidget {
  const WalletStripeStatusLine({
    super.key,
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
      color = AppPalette.green500;
      label = 'Stripe account ready — payouts enabled';
    } else if (hasAccount) {
      icon = Icons.warning_amber_rounded;
      color = AppPalette.amber500;
      label = 'Stripe account verifying';
    } else {
      icon = Icons.cancel;
      color = AppPalette.red500;
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

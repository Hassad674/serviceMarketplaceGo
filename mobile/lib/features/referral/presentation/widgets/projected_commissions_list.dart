import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../wallet/domain/entities/wallet_summary_entity.dart';
import '../../domain/entities/referral_entity.dart';

/// ProjectedCommissionsList — WALLET-UNIFY Run D parity with web
/// Run C `ProjectedCommissionsList`. Replaces the misleading
/// "0 €" sub-line shown on each per-mission row when the apporteur
/// has not yet been paid. Renders one entry per commission row
/// scoped to the attribution plus an optional synthetic "≈ X €
/// (en séquestre)" preview line when the attribution carries
/// `escrow_commission_cents > 0`.
///
/// Status → tone matrix (mirrors the web brief):
///   - paid              → "+X € reçue"     — green
///   - pending|pending_kyc → "X € en attente" — orange
///   - failed            → "X € échouée"   — red
///   - cancelled|clawed_back → skip
///   - synthetic escrow  → "≈ X € (en séquestre)" — muted italic
class ProjectedCommissionsList extends StatelessWidget {
  const ProjectedCommissionsList({
    super.key,
    required this.commissions,
    this.escrowCents = 0,
  });

  /// Commission rows already filtered to this attribution. The widget
  /// re-filters out cancelled/clawed_back rows defensively so the
  /// caller can pass the raw list.
  final List<ReferralCommission> commissions;

  /// Total funds held in escrow for this attribution, cents. When
  /// > 0 we prepend a synthetic "≈ X € (en séquestre)" row so the
  /// apporteur sees the projected commission preview before any
  /// transfer fires.
  final int escrowCents;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final renderable = commissions
        .where((c) => c.status != 'cancelled' && c.status != 'clawed_back')
        .toList(growable: false);

    if (renderable.isEmpty && escrowCents == 0) {
      return Padding(
        padding: const EdgeInsets.only(top: 4),
        child: Text(
          l10n.referralProjectionEmpty,
          style: theme.textTheme.bodySmall?.copyWith(
            fontStyle: FontStyle.italic,
            color: theme.colorScheme.onSurface.withValues(alpha: 0.55),
          ),
        ),
      );
    }

    return Container(
      key: const ValueKey('projected-commissions-list'),
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.referralProjectionPerMilestoneTitle.toUpperCase(),
            style: theme.textTheme.labelSmall?.copyWith(
              fontWeight: FontWeight.w700,
              fontSize: 11,
              letterSpacing: 0.5,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
            ),
          ),
          const SizedBox(height: 6),
          if (escrowCents > 0) _EscrowRow(escrowCents: escrowCents),
          ...renderable.map((c) => _CommissionRow(commission: c)),
        ],
      ),
    );
  }
}

class _EscrowRow extends StatelessWidget {
  const _EscrowRow({required this.escrowCents});

  final int escrowCents;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Text(
        l10n.referralProjectionStatusEscrowed(
          formatWalletSummaryCents(escrowCents),
        ),
        key: const ValueKey('projected-commission-escrow-line'),
        style: theme.textTheme.bodySmall?.copyWith(
          fontStyle: FontStyle.italic,
          color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
          fontSize: 12,
        ),
      ),
    );
  }
}

class _CommissionRow extends StatelessWidget {
  const _CommissionRow({required this.commission});

  final ReferralCommission commission;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final accent = theme.extension<AppColors>();
    final tone = _tone(commission.status);
    Color bg;
    Color fg;
    String label;
    final amount = formatWalletSummaryCents(commission.commissionCents);
    switch (tone) {
      case WalletStatusTone.paid:
        bg = (accent?.success ?? theme.colorScheme.primary)
            .withValues(alpha: 0.16);
        fg = accent?.success ?? theme.colorScheme.primary;
        label = l10n.referralProjectionStatusPaid(amount);
        break;
      case WalletStatusTone.pending:
        bg = (accent?.warning ?? theme.colorScheme.tertiary)
            .withValues(alpha: 0.18);
        fg = accent?.warning ?? theme.colorScheme.tertiary;
        label = l10n.referralProjectionStatusPending(amount);
        break;
      case WalletStatusTone.failed:
        bg = theme.colorScheme.error.withValues(alpha: 0.14);
        fg = theme.colorScheme.error;
        label = l10n.referralProjectionStatusFailed(amount);
        break;
      case WalletStatusTone.escrowed:
        bg = theme.colorScheme.onSurface.withValues(alpha: 0.08);
        fg = theme.colorScheme.onSurface.withValues(alpha: 0.7);
        label = l10n.referralProjectionStatusEscrowed(amount);
        break;
    }
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Align(
        alignment: Alignment.centerLeft,
        child: Container(
          key: ValueKey('projected-commission-row-${commission.id}'),
          padding:
              const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
          decoration: BoxDecoration(
            color: bg,
            borderRadius: BorderRadius.circular(AppTheme.radiusFull),
          ),
          child: Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              color: fg,
              fontWeight: FontWeight.w700,
              fontSize: 11,
            ),
          ),
        ),
      ),
    );
  }

  static WalletStatusTone _tone(String status) {
    switch (status) {
      case 'paid':
        return WalletStatusTone.paid;
      case 'pending':
      case 'pending_kyc':
        return WalletStatusTone.pending;
      case 'failed':
        return WalletStatusTone.failed;
      default:
        return WalletStatusTone.escrowed;
    }
  }
}

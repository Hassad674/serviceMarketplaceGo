import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';

/// Soleil v2 — 2-card segmented payment-mode picker.
/// Mirrors the web `PaymentModeToggle` component.
class PaymentModeToggleWidget extends StatelessWidget {
  const PaymentModeToggleWidget({
    super.key,
    required this.value,
    required this.onChanged,
    this.disabled = false,
  });

  final ProposalPaymentMode value;
  final ValueChanged<ProposalPaymentMode> onChanged;
  final bool disabled;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.paymentModeLabel,
          style: SoleilTextStyles.mono.copyWith(
            color: theme.colorScheme.primary,
            fontWeight: FontWeight.w700,
            fontSize: 11,
            letterSpacing: 1.2,
          ),
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: _ModeCard(
                active: value == ProposalPaymentMode.oneTime,
                disabled: disabled,
                onTap: () => onChanged(ProposalPaymentMode.oneTime),
                label: l10n.paymentModeOneTime,
                hint: l10n.paymentModeOneTimeHint,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _ModeCard(
                active: value == ProposalPaymentMode.milestone,
                disabled: disabled,
                onTap: () => onChanged(ProposalPaymentMode.milestone),
                label: l10n.paymentModeMilestone,
                hint: l10n.paymentModeMilestoneHint,
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _ModeCard extends StatelessWidget {
  const _ModeCard({
    required this.active,
    required this.disabled,
    required this.onTap,
    required this.label,
    required this.hint,
  });

  final bool active;
  final bool disabled;
  final VoidCallback onTap;
  final String label;
  final String hint;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final borderColor = active
        ? theme.colorScheme.primary
        : theme.dividerColor;
    final bgColor = active
        ? theme.colorScheme.primaryContainer
        : theme.colorScheme.surfaceContainerLowest;
    final fgColor = active
        ? theme.colorScheme.primary
        : theme.colorScheme.onSurface;

    return InkWell(
      onTap: disabled ? null : onTap,
      borderRadius: BorderRadius.circular(AppTheme.radiusXl),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: bgColor,
          borderRadius: BorderRadius.circular(AppTheme.radiusXl),
          border: Border.all(color: borderColor),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              label,
              style: SoleilTextStyles.bodyEmphasis.copyWith(
                color: fgColor,
              ),
            ),
            const SizedBox(height: 4),
            Text(
              hint,
              style: SoleilTextStyles.caption.copyWith(
                color: active
                    ? theme.colorScheme.primary.withValues(alpha: 0.8)
                    : theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

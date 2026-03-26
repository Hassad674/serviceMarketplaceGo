import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';

/// Two selectable cards for choosing between Escrow and Invoice payment.
class ProposalPaymentSelector extends StatelessWidget {
  const ProposalPaymentSelector({
    super.key,
    required this.selected,
    required this.onChanged,
  });

  final ProposalPaymentType selected;
  final ValueChanged<ProposalPaymentType> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Row(
      children: [
        Expanded(
          child: _PaymentOption(
            icon: Icons.lock_outline,
            title: l10n.proposalEscrow,
            isSelected: selected == ProposalPaymentType.escrow,
            onTap: () => onChanged(ProposalPaymentType.escrow),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: _PaymentOption(
            icon: Icons.receipt_long_outlined,
            title: l10n.proposalInvoice,
            isSelected: selected == ProposalPaymentType.invoice,
            onTap: () => onChanged(ProposalPaymentType.invoice),
          ),
        ),
      ],
    );
  }
}

class _PaymentOption extends StatelessWidget {
  const _PaymentOption({
    required this.icon,
    required this.title,
    required this.isSelected,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final bool isSelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final appColors = theme.extension<AppColors>();

    return GestureDetector(
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        curve: Curves.easeOut,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: BoxDecoration(
          color: isSelected
              ? primary.withValues(alpha: 0.05)
              : theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(
            color: isSelected
                ? primary
                : appColors?.border ?? theme.dividerColor,
            width: isSelected ? 2 : 1,
          ),
        ),
        child: Column(
          children: [
            Icon(
              icon,
              color: isSelected
                  ? primary
                  : theme.colorScheme.onSurface.withValues(alpha: 0.5),
              size: 24,
            ),
            const SizedBox(height: 8),
            Text(
              title,
              style: theme.textTheme.bodyMedium?.copyWith(
                fontWeight: FontWeight.w600,
                color: isSelected ? primary : null,
              ),
              textAlign: TextAlign.center,
            ),
            if (isSelected)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Icon(Icons.check_circle, color: primary, size: 18),
              ),
          ],
        ),
      ),
    );
  }
}

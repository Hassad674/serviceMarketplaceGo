import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/project.dart';

/// Two selectable cards for choosing between Invoice and Escrow payment.
///
/// The selected card gets a rose border, rose-50 background, and a check icon.
class PaymentTypeSelector extends StatelessWidget {
  const PaymentTypeSelector({
    super.key,
    required this.selected,
    required this.onChanged,
  });

  final PaymentType selected;
  final ValueChanged<PaymentType> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _SectionTitle(title: l10n.paymentType),
        const SizedBox(height: 12),
        _PaymentCard(
          icon: Icons.receipt_long_outlined,
          title: l10n.invoiceBilling,
          description: l10n.invoiceBillingDesc,
          isSelected: selected == PaymentType.invoice,
          onTap: () => onChanged(PaymentType.invoice),
        ),
        const SizedBox(height: 12),
        _PaymentCard(
          icon: Icons.lock_outline,
          title: l10n.escrowPayments,
          description: l10n.escrowPaymentsDesc,
          isSelected: selected == PaymentType.escrow,
          onTap: () => onChanged(PaymentType.escrow),
        ),
      ],
    );
  }
}

class _PaymentCard extends StatelessWidget {
  const _PaymentCard({
    required this.icon,
    required this.title,
    required this.description,
    required this.isSelected,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String description;
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
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: isSelected
              ? primary.withValues(alpha: 0.05)
              : theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(AppTheme.radiusLg),
          border: Border.all(
            color: isSelected
                ? primary
                : appColors?.border ?? theme.dividerColor,
            width: isSelected ? 2 : 1,
          ),
        ),
        child: Row(
          children: [
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                color: isSelected
                    ? primary.withValues(alpha: 0.1)
                    : appColors?.muted ?? theme.disabledColor,
                borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              ),
              child: Icon(
                icon,
                color: isSelected
                    ? primary
                    : theme.colorScheme.onSurface.withValues(alpha: 0.5),
                size: 22,
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    title,
                    style: theme.textTheme.titleMedium?.copyWith(
                      color: isSelected ? primary : null,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    description,
                    style: theme.textTheme.bodySmall,
                  ),
                ],
              ),
            ),
            if (isSelected)
              Icon(Icons.check_circle, color: primary, size: 24),
          ],
        ),
      ),
    );
  }
}

/// Reusable section title used across form sections.
class _SectionTitle extends StatelessWidget {
  const _SectionTitle({required this.title});

  final String title;

  @override
  Widget build(BuildContext context) {
    return Text(
      title,
      style: Theme.of(context).textTheme.titleMedium,
    );
  }
}

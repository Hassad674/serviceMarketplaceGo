import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';

/// A +/- counter widget for selecting the number of contractors.
///
/// Displays a label, the current count, and increment/decrement buttons.
/// Minimum value is 1, maximum is 99.
class ContractorCounter extends StatelessWidget {
  const ContractorCounter({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final int value;
  final ValueChanged<int> onChanged;

  static const int _min = 1;
  static const int _max = 99;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final appColors = theme.extension<AppColors>();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.jobContractorCount, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              _CounterButton(
                icon: Icons.remove,
                onPressed: value > _min ? () => onChanged(value - 1) : null,
                primary: primary,
              ),
              const SizedBox(width: 20),
              Text(
                '$value',
                style: theme.textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(width: 20),
              _CounterButton(
                icon: Icons.add,
                onPressed: value < _max ? () => onChanged(value + 1) : null,
                primary: primary,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

/// A circular icon button used for increment/decrement.
class _CounterButton extends StatelessWidget {
  const _CounterButton({
    required this.icon,
    required this.onPressed,
    required this.primary,
  });

  final IconData icon;
  final VoidCallback? onPressed;
  final Color primary;

  @override
  Widget build(BuildContext context) {
    final isEnabled = onPressed != null;

    return GestureDetector(
      onTap: onPressed,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 200),
        curve: Curves.easeOut,
        width: 40,
        height: 40,
        decoration: BoxDecoration(
          color: isEnabled
              ? primary.withValues(alpha: 0.1)
              : Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.05),
          borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        ),
        child: Icon(
          icon,
          size: 20,
          color: isEnabled
              ? primary
              : Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.3),
        ),
      ),
    );
  }
}

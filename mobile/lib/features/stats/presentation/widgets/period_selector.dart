import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/theme/theme_text_styles.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/stats_period.dart';

/// Pill-style 7d / 30d / 90d switcher used at the top of the
/// [StatsScreen]. Soleil v2 idiom: rounded-full ghost track, corail fill
/// for the selected pill.
///
/// Stateless; the parent owns the [StatsPeriod] in a Riverpod provider
/// and pushes the new value via [onChanged]. The label formatting goes
/// through [AppLocalizations] so the FR variant says "7 j / 30 j / 90 j".
class PeriodSelector extends StatelessWidget {
  const PeriodSelector({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final StatsPeriod value;
  final ValueChanged<StatsPeriod> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Semantics(
      container: true,
      label: l10n.statsPeriodSelectorLabel,
      child: Container(
        padding: const EdgeInsets.all(4),
        decoration: BoxDecoration(
          color: appColors?.muted ?? theme.colorScheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(999),
        ),
        child: Row(
          children: [
            for (final period in StatsPeriod.values)
              Expanded(
                child: _PeriodPill(
                  selected: period == value,
                  label: _label(l10n, period),
                  onTap: () => onChanged(period),
                ),
              ),
          ],
        ),
      ),
    );
  }

  String _label(AppLocalizations l10n, StatsPeriod period) {
    switch (period) {
      case StatsPeriod.sevenDays:
        return l10n.statsPeriod7d;
      case StatsPeriod.thirtyDays:
        return l10n.statsPeriod30d;
      case StatsPeriod.ninetyDays:
        return l10n.statsPeriod90d;
    }
  }
}

class _PeriodPill extends StatelessWidget {
  const _PeriodPill({
    required this.selected,
    required this.label,
    required this.onTap,
  });

  final bool selected;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: selected
          ? theme.colorScheme.primary
          : Colors.transparent,
      borderRadius: BorderRadius.circular(999),
      child: InkWell(
        borderRadius: BorderRadius.circular(999),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 10),
          child: Center(
            child: Text(
              label,
              style: SoleilTextStyles.button.copyWith(
                color: selected
                    ? theme.colorScheme.onPrimary
                    : theme.colorScheme.onSurface,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

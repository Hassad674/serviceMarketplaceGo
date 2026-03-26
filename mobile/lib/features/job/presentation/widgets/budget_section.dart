import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';

/// Section 2: Budget and duration — project type, payment frequency,
/// rate/budget range, max hours, estimated duration, and indefinite toggle.
class BudgetSection extends StatelessWidget {
  const BudgetSection({
    super.key,
    required this.budgetType,
    required this.onBudgetTypeChanged,
    required this.paymentFrequency,
    required this.onPaymentFrequencyChanged,
    required this.minRateController,
    required this.maxRateController,
    required this.maxHoursPerWeek,
    required this.onMaxHoursChanged,
    required this.minBudgetController,
    required this.maxBudgetController,
    required this.durationController,
    required this.durationUnit,
    required this.onDurationUnitChanged,
    required this.isIndefinite,
    required this.onIndefiniteChanged,
    required this.isExpanded,
    required this.onExpansionChanged,
  });

  final BudgetType budgetType;
  final ValueChanged<BudgetType> onBudgetTypeChanged;
  final PaymentFrequency paymentFrequency;
  final ValueChanged<PaymentFrequency> onPaymentFrequencyChanged;
  final TextEditingController minRateController;
  final TextEditingController maxRateController;
  final int maxHoursPerWeek;
  final ValueChanged<int> onMaxHoursChanged;
  final TextEditingController minBudgetController;
  final TextEditingController maxBudgetController;
  final TextEditingController durationController;
  final DurationUnit durationUnit;
  final ValueChanged<DurationUnit> onDurationUnitChanged;
  final bool isIndefinite;
  final ValueChanged<bool> onIndefiniteChanged;
  final bool isExpanded;
  final ValueChanged<bool> onExpansionChanged;

  /// Returns the suffix label for rate fields based on frequency.
  String _rateSuffix(AppLocalizations l10n) {
    switch (paymentFrequency) {
      case PaymentFrequency.hourly:
        return '/h';
      case PaymentFrequency.weekly:
        return '/${l10n.jobWeeks.substring(0, 3)}';
      case PaymentFrequency.monthly:
        return '/${l10n.jobMonths.substring(0, 3)}';
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return _buildExpandableContainer(
      context: context,
      theme: theme,
      appColors: appColors,
      title: l10n.jobBudgetAndDuration,
      icon: Icons.attach_money,
      isExpanded: isExpanded,
      onExpansionChanged: onExpansionChanged,
      children: [
        // Budget type: Ongoing / One-time
        Text(l10n.jobBudgetType, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        SizedBox(
          width: double.infinity,
          child: SegmentedButton<BudgetType>(
            segments: [
              ButtonSegment(
                value: BudgetType.ongoing,
                label: Text(l10n.jobOngoing),
                icon: const Icon(Icons.repeat, size: 18),
              ),
              ButtonSegment(
                value: BudgetType.oneTime,
                label: Text(l10n.jobOneTime),
                icon: const Icon(Icons.looks_one_outlined, size: 18),
              ),
            ],
            selected: {budgetType},
            onSelectionChanged: (set) => onBudgetTypeChanged(set.first),
            style: ButtonStyle(
              shape: WidgetStatePropertyAll(
                RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ),
        ),
        const SizedBox(height: 20),

        // Conditional sections based on budget type
        if (budgetType == BudgetType.ongoing)
          _OngoingBudgetFields(
            paymentFrequency: paymentFrequency,
            onPaymentFrequencyChanged: onPaymentFrequencyChanged,
            minRateController: minRateController,
            maxRateController: maxRateController,
            maxHoursPerWeek: maxHoursPerWeek,
            onMaxHoursChanged: onMaxHoursChanged,
            rateSuffix: _rateSuffix(l10n),
          )
        else
          _OneTimeBudgetFields(
            minBudgetController: minBudgetController,
            maxBudgetController: maxBudgetController,
          ),

        const SizedBox(height: 20),

        // Duration fields (shared)
        _DurationFields(
          durationController: durationController,
          durationUnit: durationUnit,
          onDurationUnitChanged: onDurationUnitChanged,
          isIndefinite: isIndefinite,
          onIndefiniteChanged: onIndefiniteChanged,
        ),
      ],
    );
  }

  Widget _buildExpandableContainer({
    required BuildContext context,
    required ThemeData theme,
    required AppColors? appColors,
    required String title,
    required IconData icon,
    required bool isExpanded,
    required ValueChanged<bool> onExpansionChanged,
    required List<Widget> children,
  }) {
    final primary = theme.colorScheme.primary;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeOut,
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(
          color: isExpanded
              ? primary.withValues(alpha: 0.3)
              : appColors?.border ?? theme.dividerColor,
        ),
        boxShadow: isExpanded ? AppTheme.cardShadow : null,
      ),
      child: Column(
        children: [
          InkWell(
            onTap: () => onExpansionChanged(!isExpanded),
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  Container(
                    width: 40,
                    height: 40,
                    decoration: BoxDecoration(
                      color: primary.withValues(alpha: 0.1),
                      borderRadius:
                          BorderRadius.circular(AppTheme.radiusSm),
                    ),
                    child: Icon(icon, color: primary, size: 20),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      title,
                      style: theme.textTheme.titleMedium,
                    ),
                  ),
                  AnimatedRotation(
                    turns: isExpanded ? 0.5 : 0,
                    duration: const Duration(milliseconds: 200),
                    child: Icon(
                      Icons.keyboard_arrow_down,
                      color: appColors?.mutedForeground ??
                          theme.colorScheme.onSurface,
                    ),
                  ),
                ],
              ),
            ),
          ),
          AnimatedCrossFade(
            firstChild: const SizedBox.shrink(),
            secondChild: Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: children,
              ),
            ),
            crossFadeState: isExpanded
                ? CrossFadeState.showSecond
                : CrossFadeState.showFirst,
            duration: const Duration(milliseconds: 200),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Ongoing budget fields — frequency, rate range, max hours
// ---------------------------------------------------------------------------

class _OngoingBudgetFields extends StatelessWidget {
  const _OngoingBudgetFields({
    required this.paymentFrequency,
    required this.onPaymentFrequencyChanged,
    required this.minRateController,
    required this.maxRateController,
    required this.maxHoursPerWeek,
    required this.onMaxHoursChanged,
    required this.rateSuffix,
  });

  final PaymentFrequency paymentFrequency;
  final ValueChanged<PaymentFrequency> onPaymentFrequencyChanged;
  final TextEditingController minRateController;
  final TextEditingController maxRateController;
  final int maxHoursPerWeek;
  final ValueChanged<int> onMaxHoursChanged;
  final String rateSuffix;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Payment frequency
        Text(
          l10n.jobPaymentFrequency,
          style: theme.textTheme.titleMedium,
        ),
        const SizedBox(height: 12),
        SizedBox(
          width: double.infinity,
          child: SegmentedButton<PaymentFrequency>(
            segments: [
              ButtonSegment(
                value: PaymentFrequency.hourly,
                label: Text(l10n.jobHourly),
              ),
              ButtonSegment(
                value: PaymentFrequency.weekly,
                label: Text(l10n.jobWeekly),
              ),
              ButtonSegment(
                value: PaymentFrequency.monthly,
                label: Text(l10n.jobMonthly),
              ),
            ],
            selected: {paymentFrequency},
            onSelectionChanged: (set) =>
                onPaymentFrequencyChanged(set.first),
            style: ButtonStyle(
              shape: WidgetStatePropertyAll(
                RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ),
        ),
        const SizedBox(height: 16),

        // Rate range
        Row(
          children: [
            Expanded(
              child: TextFormField(
                controller: minRateController,
                decoration: InputDecoration(
                  labelText: l10n.jobMinRate,
                  prefixText: '\u20AC ',
                  suffixText: rateSuffix,
                ),
                keyboardType: TextInputType.number,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: TextFormField(
                controller: maxRateController,
                decoration: InputDecoration(
                  labelText: l10n.jobMaxRate,
                  prefixText: '\u20AC ',
                  suffixText: rateSuffix,
                ),
                keyboardType: TextInputType.number,
              ),
            ),
          ],
        ),

        // Max hours/week (only for hourly)
        if (paymentFrequency == PaymentFrequency.hourly) ...[
          const SizedBox(height: 16),
          _HoursCounter(
            label: l10n.jobMaxHours,
            value: maxHoursPerWeek,
            onChanged: onMaxHoursChanged,
          ),
        ],
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// One-time budget fields — budget min/max
// ---------------------------------------------------------------------------

class _OneTimeBudgetFields extends StatelessWidget {
  const _OneTimeBudgetFields({
    required this.minBudgetController,
    required this.maxBudgetController,
  });

  final TextEditingController minBudgetController;
  final TextEditingController maxBudgetController;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Row(
      children: [
        Expanded(
          child: TextFormField(
            controller: minBudgetController,
            decoration: InputDecoration(
              labelText: l10n.jobMinBudget,
              prefixText: '\u20AC ',
            ),
            keyboardType: TextInputType.number,
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: TextFormField(
            controller: maxBudgetController,
            decoration: InputDecoration(
              labelText: l10n.jobMaxBudget,
              prefixText: '\u20AC ',
            ),
            keyboardType: TextInputType.number,
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Duration fields — estimated duration + indefinite toggle
// ---------------------------------------------------------------------------

class _DurationFields extends StatelessWidget {
  const _DurationFields({
    required this.durationController,
    required this.durationUnit,
    required this.onDurationUnitChanged,
    required this.isIndefinite,
    required this.onIndefiniteChanged,
  });

  final TextEditingController durationController;
  final DurationUnit durationUnit;
  final ValueChanged<DurationUnit> onDurationUnitChanged;
  final bool isIndefinite;
  final ValueChanged<bool> onIndefiniteChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Indefinite toggle
        Row(
          children: [
            Expanded(
              child: Text(
                l10n.jobIndefinite,
                style: theme.textTheme.bodyMedium,
              ),
            ),
            Switch(
              value: isIndefinite,
              onChanged: onIndefiniteChanged,
              activeThumbColor: theme.colorScheme.primary,
            ),
          ],
        ),

        // Duration input (hidden when indefinite)
        if (!isIndefinite) ...[
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                flex: 2,
                child: TextFormField(
                  controller: durationController,
                  decoration: InputDecoration(
                    labelText: l10n.jobEstimatedDuration,
                  ),
                  keyboardType: TextInputType.number,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                flex: 3,
                child: DropdownButtonFormField<DurationUnit>(
                  initialValue: durationUnit,
                  decoration: const InputDecoration(
                    contentPadding: EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 16,
                    ),
                  ),
                  items: [
                    DropdownMenuItem(
                      value: DurationUnit.weeks,
                      child: Text(l10n.jobWeeks),
                    ),
                    DropdownMenuItem(
                      value: DurationUnit.months,
                      child: Text(l10n.jobMonths),
                    ),
                  ],
                  onChanged: (value) {
                    if (value != null) onDurationUnitChanged(value);
                  },
                ),
              ),
            ],
          ),
        ],
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Hours counter — small inline counter for max hours/week
// ---------------------------------------------------------------------------

class _HoursCounter extends StatelessWidget {
  const _HoursCounter({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final int value;
  final ValueChanged<int> onChanged;

  static const int _min = 1;
  static const int _max = 80;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final appColors = theme.extension<AppColors>();

    return Row(
      children: [
        Expanded(
          child: Text(label, style: theme.textTheme.bodyMedium),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            color: theme.colorScheme.surface,
            borderRadius: BorderRadius.circular(AppTheme.radiusSm),
            border: Border.all(
              color: appColors?.border ?? theme.dividerColor,
            ),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              _SmallButton(
                icon: Icons.remove,
                onPressed:
                    value > _min ? () => onChanged(value - 1) : null,
                primary: primary,
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: Text(
                  '$value',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
              _SmallButton(
                icon: Icons.add,
                onPressed:
                    value < _max ? () => onChanged(value + 1) : null,
                primary: primary,
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _SmallButton extends StatelessWidget {
  const _SmallButton({
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
      child: Container(
        width: 32,
        height: 32,
        decoration: BoxDecoration(
          color: isEnabled
              ? primary.withValues(alpha: 0.1)
              : Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.05),
          borderRadius: BorderRadius.circular(6),
        ),
        child: Icon(
          icon,
          size: 16,
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

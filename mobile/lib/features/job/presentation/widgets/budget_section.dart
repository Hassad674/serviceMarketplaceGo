import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';

/// Section 2: Budget — project type (one-shot / long-term) and budget range.
class BudgetSection extends StatelessWidget {
  const BudgetSection({
    super.key,
    required this.budgetType,
    required this.onBudgetTypeChanged,
    required this.minBudgetController,
    required this.maxBudgetController,
    required this.isExpanded,
    required this.onExpansionChanged,
  });

  final BudgetType budgetType;
  final ValueChanged<BudgetType> onBudgetTypeChanged;
  final TextEditingController minBudgetController;
  final TextEditingController maxBudgetController;
  final bool isExpanded;
  final ValueChanged<bool> onExpansionChanged;

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
        // Budget type: One-shot / Long-term
        Text(l10n.jobBudgetType, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        SizedBox(
          width: double.infinity,
          child: SegmentedButton<BudgetType>(
            segments: [
              ButtonSegment(
                value: BudgetType.oneShot,
                label: Text(l10n.jobOneTime),
                icon: const Icon(Icons.looks_one_outlined, size: 18),
              ),
              ButtonSegment(
                value: BudgetType.longTerm,
                label: Text(l10n.jobOngoing),
                icon: const Icon(Icons.repeat, size: 18),
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

        // Min / Max budget
        Row(
          children: [
            Expanded(
              child: TextFormField(
                controller: minBudgetController,
                decoration: InputDecoration(
                  labelText: l10n.jobMinBudget,
                  prefixText: '\u20AC ',
                ),
                keyboardType: TextInputType.number,
                validator: (value) {
                  if (value == null || value.trim().isEmpty) {
                    return l10n.fieldRequired;
                  }
                  final parsed = int.tryParse(value.trim());
                  if (parsed == null || parsed <= 0) {
                    return l10n.fieldRequired;
                  }
                  return null;
                },
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
                validator: (value) {
                  if (value == null || value.trim().isEmpty) {
                    return l10n.fieldRequired;
                  }
                  final parsed = int.tryParse(value.trim());
                  if (parsed == null || parsed <= 0) {
                    return l10n.fieldRequired;
                  }
                  return null;
                },
              ),
            ),
          ],
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

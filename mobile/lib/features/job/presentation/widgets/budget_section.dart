import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/job.dart';

/// M-09 — Soleil v2 budget & duration section.
///
/// Public prop interface (`budgetType`, `onBudgetTypeChanged`,
/// `minBudgetController`, `maxBudgetController`, `isExpanded`,
/// `onExpansionChanged`) is intentionally unchanged. Behaviour, validators
/// and controllers are untouched — only the visual identity ports to ivoire
/// & corail with mono labels and rounded segmented pills.
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

    return _SoleilSectionCard(
      title: l10n.jobBudgetAndDuration,
      number: 2,
      isExpanded: isExpanded,
      onExpansionChanged: onExpansionChanged,
      children: [
        _MonoLabel(text: l10n.jobBudgetType),
        const SizedBox(height: 10),
        _BudgetTypePicker(
          value: budgetType,
          onChanged: onBudgetTypeChanged,
        ),
        const SizedBox(height: 22),
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(
              child: _BudgetField(
                controller: minBudgetController,
                label: l10n.jobMinBudget,
                fieldRequiredLabel: l10n.fieldRequired,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _BudgetField(
                controller: maxBudgetController,
                label: l10n.jobMaxBudget,
                fieldRequiredLabel: l10n.fieldRequired,
              ),
            ),
          ],
        ),
        if (appColors == null) const SizedBox.shrink(),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Soleil pill toggle for budget type
// ---------------------------------------------------------------------------

class _BudgetTypePicker extends StatelessWidget {
  const _BudgetTypePicker({required this.value, required this.onChanged});

  final BudgetType value;
  final ValueChanged<BudgetType> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final border = appColors?.border ?? theme.colorScheme.outline;
    final bg = theme.colorScheme.surface;

    return Container(
      padding: const EdgeInsets.all(4),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        border: Border.all(color: border),
      ),
      child: Row(
        children: [
          Expanded(
            child: _BudgetTypePill(
              label: l10n.jobOneTime,
              icon: Icons.bolt_outlined,
              selected: value == BudgetType.oneShot,
              onTap: () => onChanged(BudgetType.oneShot),
            ),
          ),
          Expanded(
            child: _BudgetTypePill(
              label: l10n.jobOngoing,
              icon: Icons.sync,
              selected: value == BudgetType.longTerm,
              onTap: () => onChanged(BudgetType.longTerm),
            ),
          ),
        ],
      ),
    );
  }
}

class _BudgetTypePill extends StatelessWidget {
  const _BudgetTypePill({
    required this.label,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 180),
          curve: Curves.easeOut,
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: selected ? primary : Colors.transparent,
            borderRadius: BorderRadius.circular(AppTheme.radiusFull),
          ),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                icon,
                size: 16,
                color: selected ? theme.colorScheme.onPrimary : mute,
              ),
              const SizedBox(width: 6),
              Flexible(
                child: Text(
                  label,
                  textAlign: TextAlign.center,
                  overflow: TextOverflow.ellipsis,
                  style: SoleilTextStyles.button.copyWith(
                    color: selected ? theme.colorScheme.onPrimary : mute,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Currency input field with euro suffix and mono numerals
// ---------------------------------------------------------------------------

class _BudgetField extends StatelessWidget {
  const _BudgetField({
    required this.controller,
    required this.label,
    required this.fieldRequiredLabel,
  });

  final TextEditingController controller;
  final String label;
  final String fieldRequiredLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _MonoLabel(text: label),
        const SizedBox(height: 8),
        TextFormField(
          controller: controller,
          keyboardType: TextInputType.number,
          style: SoleilTextStyles.bodyLarge.copyWith(color: theme.colorScheme.onSurface),
          decoration: InputDecoration(
            suffixText: '€',
            suffixStyle: SoleilTextStyles.body.copyWith(color: mute),
          ),
          validator: (value) {
            if (value == null || value.trim().isEmpty) {
              return fieldRequiredLabel;
            }
            final parsed = int.tryParse(value.trim());
            if (parsed == null || parsed <= 0) {
              return fieldRequiredLabel;
            }
            return null;
          },
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Mono uppercase label (Soleil signature)
// ---------------------------------------------------------------------------

class _MonoLabel extends StatelessWidget {
  const _MonoLabel({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Text(
      text.toUpperCase(),
      style: SoleilTextStyles.mono.copyWith(
        color: mute,
        fontSize: 11,
        fontWeight: FontWeight.w700,
        letterSpacing: 0.8,
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Soleil section card with corail badge + chevron toggle
// ---------------------------------------------------------------------------

class _SoleilSectionCard extends StatelessWidget {
  const _SoleilSectionCard({
    required this.title,
    required this.number,
    required this.isExpanded,
    required this.onExpansionChanged,
    required this.children,
  });

  final String title;
  final int number;
  final bool isExpanded;
  final ValueChanged<bool> onExpansionChanged;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final accentSoft = appColors?.accentSoft ?? theme.colorScheme.primaryContainer;
    final primaryDeep = appColors?.primaryDeep ?? primary;
    final border = appColors?.border ?? theme.colorScheme.outline;
    final borderStrong = appColors?.borderStrong ?? theme.colorScheme.outline;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeOut,
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(AppTheme.radius2xl),
        border: Border.all(
          color: isExpanded ? borderStrong : border,
        ),
        boxShadow: isExpanded ? AppTheme.cardShadow : null,
      ),
      child: Column(
        children: [
          InkWell(
            onTap: () => onExpansionChanged(!isExpanded),
            borderRadius: BorderRadius.circular(AppTheme.radius2xl),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(20, 18, 20, 18),
              child: Row(
                children: [
                  Container(
                    width: 32,
                    height: 32,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      color: isExpanded ? accentSoft : theme.colorScheme.surface,
                      border: Border.all(
                        color: isExpanded ? primary : borderStrong,
                      ),
                    ),
                    child: Text(
                      '$number',
                      style: SoleilTextStyles.mono.copyWith(
                        color: isExpanded ? primaryDeep : mute,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Text(
                      title,
                      style: SoleilTextStyles.titleMedium,
                    ),
                  ),
                  AnimatedRotation(
                    turns: isExpanded ? 0.25 : 0,
                    duration: const Duration(milliseconds: 200),
                    child: Icon(
                      Icons.chevron_right,
                      color: mute,
                    ),
                  ),
                ],
              ),
            ),
          ),
          AnimatedCrossFade(
            firstChild: const SizedBox.shrink(),
            secondChild: Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 22),
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

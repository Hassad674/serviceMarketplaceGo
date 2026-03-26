import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/project.dart';

/// Escrow-mode structure section: toggle between Milestone and One-time,
/// then show the appropriate sub-form.
class EscrowStructureSection extends StatelessWidget {
  const EscrowStructureSection({
    super.key,
    required this.structure,
    required this.onStructureChanged,
    required this.milestones,
    required this.amount,
    required this.onAmountChanged,
    required this.onMilestoneAdded,
    required this.onMilestoneRemoved,
    required this.onMilestoneChanged,
  });

  final ProjectStructure structure;
  final ValueChanged<ProjectStructure> onStructureChanged;
  final List<MilestoneData> milestones;
  final double amount;
  final ValueChanged<double> onAmountChanged;
  final VoidCallback onMilestoneAdded;
  final ValueChanged<int> onMilestoneRemoved;
  final VoidCallback onMilestoneChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.projectStructure, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        SegmentedButton<ProjectStructure>(
          segments: [
            ButtonSegment(
              value: ProjectStructure.milestone,
              label: Text(l10n.milestone),
              icon: const Icon(Icons.flag_outlined, size: 18),
            ),
            ButtonSegment(
              value: ProjectStructure.oneTime,
              label: Text(l10n.oneTime),
              icon: const Icon(Icons.payments_outlined, size: 18),
            ),
          ],
          selected: {structure},
          onSelectionChanged: (set) => onStructureChanged(set.first),
          style: ButtonStyle(
            shape: WidgetStatePropertyAll(
              RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              ),
            ),
          ),
        ),
        const SizedBox(height: 16),
        if (structure == ProjectStructure.milestone)
          _MilestoneList(
            milestones: milestones,
            onAdded: onMilestoneAdded,
            onRemoved: onMilestoneRemoved,
            onChanged: onMilestoneChanged,
          )
        else
          _OneTimeAmount(
            amount: amount,
            onChanged: onAmountChanged,
          ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Milestone list
// ---------------------------------------------------------------------------

class _MilestoneList extends StatelessWidget {
  const _MilestoneList({
    required this.milestones,
    required this.onAdded,
    required this.onRemoved,
    required this.onChanged,
  });

  final List<MilestoneData> milestones;
  final VoidCallback onAdded;
  final ValueChanged<int> onRemoved;
  final VoidCallback onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      children: [
        for (int i = 0; i < milestones.length; i++)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: _MilestoneCard(
              index: i,
              milestone: milestones[i],
              canDelete: milestones.length > 1,
              onDelete: () => onRemoved(i),
              onChanged: onChanged,
            ),
          ),
        SizedBox(
          width: double.infinity,
          child: OutlinedButton.icon(
            onPressed: onAdded,
            icon: const Icon(Icons.add, size: 18),
            label: Text(l10n.addMilestone),
            style: OutlinedButton.styleFrom(
              foregroundColor: theme.colorScheme.primary,
              side: BorderSide(
                color: theme.colorScheme.primary.withValues(alpha: 0.3),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Single milestone card
// ---------------------------------------------------------------------------

class _MilestoneCard extends StatelessWidget {
  const _MilestoneCard({
    required this.index,
    required this.milestone,
    required this.canDelete,
    required this.onDelete,
    required this.onChanged,
  });

  final int index;
  final MilestoneData milestone;
  final bool canDelete;
  final VoidCallback onDelete;
  final VoidCallback onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: appColors?.border ?? theme.dividerColor,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Text(
                '${l10n.milestone} ${index + 1}',
                style: theme.textTheme.titleMedium,
              ),
              const Spacer(),
              if (canDelete)
                IconButton(
                  onPressed: onDelete,
                  icon: const Icon(Icons.delete_outline, size: 20),
                  color: theme.colorScheme.error,
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(
                    minWidth: 32,
                    minHeight: 32,
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          TextFormField(
            initialValue: milestone.title,
            decoration: InputDecoration(
              labelText: l10n.milestoneTitle,
            ),
            onChanged: (value) {
              milestone.title = value;
              onChanged();
            },
          ),
          const SizedBox(height: 12),
          TextFormField(
            initialValue: milestone.description,
            decoration: InputDecoration(
              labelText: l10n.milestoneDescription,
            ),
            maxLines: 3,
            onChanged: (value) {
              milestone.description = value;
              onChanged();
            },
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: TextFormField(
                  initialValue: milestone.amount > 0
                      ? milestone.amount.toStringAsFixed(0)
                      : '',
                  decoration: InputDecoration(
                    labelText: l10n.milestoneAmount,
                    prefixText: '\u20AC ',
                  ),
                  keyboardType: TextInputType.number,
                  onChanged: (value) {
                    milestone.amount = double.tryParse(value) ?? 0;
                    onChanged();
                  },
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: _DatePickerField(
                  label: l10n.deadline,
                  value: milestone.deadline,
                  onChanged: (date) {
                    milestone.deadline = date;
                    onChanged();
                  },
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// One-time amount
// ---------------------------------------------------------------------------

class _OneTimeAmount extends StatelessWidget {
  const _OneTimeAmount({
    required this.amount,
    required this.onChanged,
  });

  final double amount;
  final ValueChanged<double> onChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return TextFormField(
      initialValue: amount > 0 ? amount.toStringAsFixed(0) : '',
      decoration: InputDecoration(
        labelText: l10n.totalAmount,
        prefixText: '\u20AC ',
      ),
      keyboardType: TextInputType.number,
      onChanged: (value) => onChanged(double.tryParse(value) ?? 0),
    );
  }
}

// ---------------------------------------------------------------------------
// Reusable date picker field
// ---------------------------------------------------------------------------

class _DatePickerField extends StatelessWidget {
  const _DatePickerField({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final DateTime? value;
  final ValueChanged<DateTime?> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return GestureDetector(
      onTap: () => _pickDate(context),
      child: AbsorbPointer(
        child: TextFormField(
          decoration: InputDecoration(
            labelText: label,
            suffixIcon: const Icon(Icons.calendar_today_outlined, size: 18),
          ),
          controller: TextEditingController(
            text: value != null
                ? '${value!.day.toString().padLeft(2, '0')}/${value!.month.toString().padLeft(2, '0')}/${value!.year}'
                : '',
          ),
          style: theme.textTheme.bodyMedium,
        ),
      ),
    );
  }

  Future<void> _pickDate(BuildContext context) async {
    final now = DateTime.now();
    final picked = await showDatePicker(
      context: context,
      initialDate: value ?? now,
      firstDate: now,
      lastDate: now.add(const Duration(days: 730)),
    );
    if (picked != null) {
      onChanged(picked);
    }
  }
}

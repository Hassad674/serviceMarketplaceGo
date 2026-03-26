import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/proposal.dart';

/// Editable card for a single milestone within the proposal form.
class ProposalMilestoneCard extends StatelessWidget {
  const ProposalMilestoneCard({
    super.key,
    required this.index,
    required this.milestone,
    required this.canDelete,
    required this.onDelete,
    required this.onChanged,
  });

  final int index;
  final ProposalMilestone milestone;
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
          _buildHeader(l10n, theme),
          const SizedBox(height: 12),
          _buildTitleField(l10n),
          const SizedBox(height: 12),
          _buildDescriptionField(l10n),
          const SizedBox(height: 12),
          _buildAmountAndDeadline(l10n, theme),
        ],
      ),
    );
  }

  Widget _buildHeader(AppLocalizations l10n, ThemeData theme) {
    return Row(
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
            constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
          ),
      ],
    );
  }

  Widget _buildTitleField(AppLocalizations l10n) {
    return TextFormField(
      initialValue: milestone.title,
      decoration: InputDecoration(labelText: l10n.milestoneTitle),
      onChanged: (value) {
        milestone.title = value;
        onChanged();
      },
    );
  }

  Widget _buildDescriptionField(AppLocalizations l10n) {
    return TextFormField(
      initialValue: milestone.description,
      decoration: InputDecoration(labelText: l10n.milestoneDescription),
      maxLines: 2,
      onChanged: (value) {
        milestone.description = value;
        onChanged();
      },
    );
  }

  Widget _buildAmountAndDeadline(AppLocalizations l10n, ThemeData theme) {
    return Row(
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
          child: _DeadlineField(
            label: l10n.deadline,
            value: milestone.deadline,
            onChanged: (date) {
              milestone.deadline = date;
              onChanged();
            },
          ),
        ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Date picker field (scoped to this file)
// ---------------------------------------------------------------------------

class _DeadlineField extends StatelessWidget {
  const _DeadlineField({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final DateTime? value;
  final ValueChanged<DateTime?> onChanged;

  String _formatDate(DateTime date) {
    final d = date.day.toString().padLeft(2, '0');
    final m = date.month.toString().padLeft(2, '0');
    return '$d/$m/${date.year}';
  }

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => _pickDate(context),
      child: AbsorbPointer(
        child: TextFormField(
          decoration: InputDecoration(
            labelText: label,
            suffixIcon: const Icon(Icons.calendar_today_outlined, size: 18),
          ),
          controller: TextEditingController(
            text: value != null ? _formatDate(value!) : '',
          ),
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

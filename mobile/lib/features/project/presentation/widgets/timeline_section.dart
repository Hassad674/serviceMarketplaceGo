// ignore_for_file: deprecated_member_use
import 'package:flutter/material.dart';

import '../../../../l10n/app_localizations.dart';

/// Section 4: Start date, deadline (or ongoing toggle).
class TimelineSection extends StatelessWidget {
  const TimelineSection({
    super.key,
    required this.startDate,
    required this.deadline,
    required this.ongoing,
    required this.onStartDateChanged,
    required this.onDeadlineChanged,
    required this.onOngoingChanged,
  });

  final DateTime? startDate;
  final DateTime? deadline;
  final bool ongoing;
  final ValueChanged<DateTime?> onStartDateChanged;
  final ValueChanged<DateTime?> onDeadlineChanged;
  final ValueChanged<bool> onOngoingChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.timeline, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),

        // Start date
        _DateField(
          label: l10n.startDate,
          value: startDate,
          onChanged: onStartDateChanged,
        ),
        const SizedBox(height: 12),

        // Ongoing toggle
        Row(
          children: [
            Expanded(
              child: Text(
                l10n.ongoing,
                style: theme.textTheme.bodyMedium,
              ),
            ),
            Switch(
              value: ongoing,
              onChanged: onOngoingChanged,
              activeColor: theme.colorScheme.primary,
            ),
          ],
        ),

        // Deadline (hidden when ongoing)
        if (!ongoing) ...[
          const SizedBox(height: 12),
          _DateField(
            label: l10n.deadline,
            value: deadline,
            onChanged: onDeadlineChanged,
          ),
        ],
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Reusable date picker field
// ---------------------------------------------------------------------------

class _DateField extends StatelessWidget {
  const _DateField({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final DateTime? value;
  final ValueChanged<DateTime?> onChanged;

  String _formatDate(DateTime date) {
    return '${date.day.toString().padLeft(2, '0')}/'
        '${date.month.toString().padLeft(2, '0')}/'
        '${date.year}';
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

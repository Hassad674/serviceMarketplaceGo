import 'package:flutter/material.dart';

import '../../domain/entities/referral_entity.dart';

const Map<String, String> _sizeLabels = {
  'tpe': 'TPE (< 10 employees)',
  'pme': 'SME (10-250 employees)',
  'eti': 'Mid-cap (250-5000)',
  'ge': 'Large (> 5000)',
};

/// Renders the safe-to-reveal client attributes for the provider's
/// modal-as-screen view. Company name / logo / contact are absent.
class AnonymizedClientCard extends StatelessWidget {
  const AnonymizedClientCard({super.key, required this.snapshot});

  final ClientSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        side: BorderSide(color: theme.colorScheme.outlineVariant),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(
                  radius: 22,
                  backgroundColor: theme.colorScheme.secondaryContainer,
                  child: Icon(
                    Icons.business,
                    color: theme.colorScheme.onSecondaryContainer,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Proposed client',
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      Text(
                        'Identity revealed on acceptance',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            if (snapshot.isEmpty)
              Text(
                'The referrer chose not to reveal any details before acceptance.',
                style: theme.textTheme.bodySmall,
              )
            else
              _buildFields(theme),
          ],
        ),
      ),
    );
  }

  Widget _buildFields(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (snapshot.industry != null && snapshot.industry!.isNotEmpty)
          _Row(label: 'Industry', value: snapshot.industry!),
        if (snapshot.sizeBucket != null && snapshot.sizeBucket!.isNotEmpty)
          _Row(
            label: 'Size',
            value: _sizeLabels[snapshot.sizeBucket!] ?? snapshot.sizeBucket!,
          ),
        if (snapshot.region != null && snapshot.region!.isNotEmpty)
          _Row(label: 'Region', value: snapshot.region!),
        if (snapshot.budgetEstimateMinCents != null)
          _Row(
            label: 'Budget',
            value: _formatBudget(
              snapshot.budgetEstimateMinCents,
              snapshot.budgetEstimateMaxCents,
              snapshot.budgetCurrency,
            ),
          ),
        if (snapshot.timeline != null && snapshot.timeline!.isNotEmpty)
          _Row(label: 'Timing', value: snapshot.timeline!),
        if (snapshot.needSummary != null && snapshot.needSummary!.isNotEmpty) ...[
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'NEED',
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 4),
                Text(snapshot.needSummary!, style: theme.textTheme.bodyMedium),
              ],
            ),
          ),
        ],
      ],
    );
  }

  static String _formatBudget(int? minCents, int? maxCents, String? currency) {
    if (minCents == null) return '';
    final cur = (currency ?? 'EUR').toUpperCase();
    final min = (minCents / 100).toStringAsFixed(0);
    final max = maxCents != null ? (maxCents / 100).toStringAsFixed(0) : null;
    if (max != null && max != min) return '$min – $max $cur';
    return '$min $cur';
  }
}

class _Row extends StatelessWidget {
  const _Row({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 90,
            child: Text(
              label,
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
          Expanded(child: Text(value, style: theme.textTheme.bodyMedium)),
        ],
      ),
    );
  }
}

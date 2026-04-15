import 'package:flutter/material.dart';

import '../../domain/entities/referral_entity.dart';

/// Renders the safe-to-reveal provider attributes for the client viewing
/// a pre-active referral. Identity / name / photo are absent on purpose
/// (Modèle A confidentiality). Empty fields collapse silently so the
/// card adapts to whatever the apporteur chose to reveal.
class AnonymizedProviderCard extends StatelessWidget {
  const AnonymizedProviderCard({super.key, required this.snapshot});

  final ProviderSnapshot snapshot;

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
                  backgroundColor: theme.colorScheme.primaryContainer,
                  child: Icon(
                    Icons.auto_awesome,
                    color: theme.colorScheme.onPrimaryContainer,
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Recommended provider',
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
        if (snapshot.expertiseDomains.isNotEmpty)
          _Row(label: 'Expertise', value: snapshot.expertiseDomains.join(', ')),
        if (snapshot.yearsExperience != null)
          _Row(label: 'Experience', value: '${snapshot.yearsExperience} years'),
        if (snapshot.averageRating != null)
          _Row(
            label: 'Rating',
            value:
                '${snapshot.averageRating!.toStringAsFixed(1)} / 5${snapshot.reviewCount != null ? ' (${snapshot.reviewCount} reviews)' : ''}',
          ),
        if (snapshot.pricingMinCents != null)
          _Row(
            label: 'Rate',
            value: _formatPriceRange(
              snapshot.pricingMinCents,
              snapshot.pricingMaxCents,
              snapshot.pricingCurrency,
              snapshot.pricingType,
            ),
          ),
        if (snapshot.region != null && snapshot.region!.isNotEmpty)
          _Row(label: 'Region', value: snapshot.region!),
        if (snapshot.languages.isNotEmpty)
          _Row(label: 'Languages', value: snapshot.languages.join(', ').toUpperCase()),
        if (snapshot.availabilityState != null && snapshot.availabilityState!.isNotEmpty)
          _Row(label: 'Availability', value: snapshot.availabilityState!),
      ],
    );
  }

  static String _formatPriceRange(int? minCents, int? maxCents, String? currency, String? pricingType) {
    if (minCents == null) return '';
    final cur = (currency ?? 'EUR').toUpperCase();
    final min = (minCents / 100).toStringAsFixed(0);
    final max = maxCents != null ? (maxCents / 100).toStringAsFixed(0) : null;
    final suffix = pricingType == 'daily' ? ' /d' : pricingType == 'hourly' ? ' /h' : '';
    if (max != null && max != min) return '$min – $max $cur$suffix';
    return '$min $cur$suffix';
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
            width: 100,
            child: Text(
              label,
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                fontWeight: FontWeight.w500,
              ),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: theme.textTheme.bodyMedium,
            ),
          ),
        ],
      ),
    );
  }
}

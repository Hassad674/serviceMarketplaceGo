import 'package:flutter/material.dart';

/// All Stripe-supported countries for the country selector.
const stripeCountries = <String, String>{
  'AT': 'Austria',
  'AU': 'Australia',
  'BE': 'Belgium',
  'BG': 'Bulgaria',
  'BR': 'Brazil',
  'CA': 'Canada',
  'CH': 'Switzerland',
  'CY': 'Cyprus',
  'CZ': 'Czech Republic',
  'DE': 'Germany',
  'DK': 'Denmark',
  'EE': 'Estonia',
  'ES': 'Spain',
  'FI': 'Finland',
  'FR': 'France',
  'GB': 'United Kingdom',
  'GR': 'Greece',
  'HK': 'Hong Kong',
  'HR': 'Croatia',
  'HU': 'Hungary',
  'IE': 'Ireland',
  'IN': 'India',
  'IT': 'Italy',
  'JP': 'Japan',
  'LT': 'Lithuania',
  'LU': 'Luxembourg',
  'LV': 'Latvia',
  'MT': 'Malta',
  'MX': 'Mexico',
  'MY': 'Malaysia',
  'NL': 'Netherlands',
  'NO': 'Norway',
  'NZ': 'New Zealand',
  'PL': 'Poland',
  'PT': 'Portugal',
  'RO': 'Romania',
  'SE': 'Sweden',
  'SG': 'Singapore',
  'SI': 'Slovenia',
  'SK': 'Slovakia',
  'TH': 'Thailand',
  'AE': 'United Arab Emirates',
  'US': 'United States',
};

/// Country selector for the payment info form.
class CountrySelectorSection extends StatelessWidget {
  const CountrySelectorSection({
    super.key,
    required this.value,
    required this.onChanged,
  });

  final String value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.dividerColor),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.public, color: theme.colorScheme.primary),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Activity Country',
                        style: theme.textTheme.titleSmall?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 2),
                      Text(
                        'Where is your professional activity based?',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            DropdownButtonFormField<String>(
              initialValue: value.isEmpty ? null : value,
              decoration: const InputDecoration(
                labelText: 'Country',
                border: OutlineInputBorder(),
              ),
              items: stripeCountries.entries
                  .map(
                    (e) => DropdownMenuItem(
                      value: e.key,
                      child: Text(e.value),
                    ),
                  )
                  .toList(),
              onChanged: (v) {
                if (v != null) onChanged(v);
              },
            ),
          ],
        ),
      ),
    );
  }
}

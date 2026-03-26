// ignore_for_file: deprecated_member_use
import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../types/project.dart';

/// Invoice billing sub-form: billing type (Fixed/Hourly), rate, frequency.
class InvoiceBillingSection extends StatelessWidget {
  const InvoiceBillingSection({
    super.key,
    required this.billingType,
    required this.onBillingTypeChanged,
    required this.rate,
    required this.onRateChanged,
    required this.frequency,
    required this.onFrequencyChanged,
  });

  final BillingType billingType;
  final ValueChanged<BillingType> onBillingTypeChanged;
  final double rate;
  final ValueChanged<double> onRateChanged;
  final BillingFrequency frequency;
  final ValueChanged<BillingFrequency> onFrequencyChanged;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l10n.billingDetails, style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),

        // Billing type toggle
        SegmentedButton<BillingType>(
          segments: [
            ButtonSegment(
              value: BillingType.fixed,
              label: Text(l10n.fixed),
              icon: const Icon(Icons.attach_money, size: 18),
            ),
            ButtonSegment(
              value: BillingType.hourly,
              label: Text(l10n.hourly),
              icon: const Icon(Icons.schedule, size: 18),
            ),
          ],
          selected: {billingType},
          onSelectionChanged: (set) => onBillingTypeChanged(set.first),
          style: ButtonStyle(
            shape: WidgetStatePropertyAll(
              RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(AppTheme.radiusMd),
              ),
            ),
          ),
        ),
        const SizedBox(height: 16),

        // Rate input
        TextFormField(
          initialValue: rate > 0 ? rate.toStringAsFixed(0) : '',
          decoration: InputDecoration(
            labelText: l10n.rate,
            prefixText: '\u20AC ',
            suffixText: billingType == BillingType.hourly ? '/h' : '',
          ),
          keyboardType: TextInputType.number,
          onChanged: (value) => onRateChanged(double.tryParse(value) ?? 0),
        ),
        const SizedBox(height: 16),

        // Frequency dropdown
        DropdownButtonFormField<BillingFrequency>(
          value: frequency,
          decoration: InputDecoration(
            labelText: l10n.frequency,
          ),
          items: [
            DropdownMenuItem(
              value: BillingFrequency.weekly,
              child: Text(l10n.weekly),
            ),
            DropdownMenuItem(
              value: BillingFrequency.biWeekly,
              child: Text(l10n.biWeekly),
            ),
            DropdownMenuItem(
              value: BillingFrequency.monthly,
              child: Text(l10n.monthly),
            ),
          ],
          onChanged: (value) {
            if (value != null) onFrequencyChanged(value);
          },
        ),
      ],
    );
  }
}

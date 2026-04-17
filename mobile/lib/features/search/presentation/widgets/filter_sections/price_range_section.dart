/// price_range_section.dart — two number fields feeding pricingMin
/// and pricingMax. Both bounds are optional; the backend handles
/// single-bound pricing clauses correctly.
library;

import 'package:flutter/material.dart';

import 'filter_primitives.dart';

class PriceRangeSection extends StatelessWidget {
  const PriceRangeSection({
    super.key,
    required this.priceMin,
    required this.priceMax,
    required this.onPriceMinChanged,
    required this.onPriceMaxChanged,
    required this.sectionTitle,
    required this.minLabel,
    required this.maxLabel,
  });

  final int? priceMin;
  final int? priceMax;
  final ValueChanged<int?> onPriceMinChanged;
  final ValueChanged<int?> onPriceMaxChanged;
  final String sectionTitle;
  final String minLabel;
  final String maxLabel;

  @override
  Widget build(BuildContext context) {
    return FilterSectionShell(
      title: sectionTitle,
      child: Row(
        children: [
          Expanded(
            child: FilterNumberField(
              key: const ValueKey('price-min'),
              value: priceMin,
              onChanged: onPriceMinChanged,
              label: minLabel,
              semanticsLabel: minLabel,
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: FilterNumberField(
              key: const ValueKey('price-max'),
              value: priceMax,
              onChanged: onPriceMaxChanged,
              label: maxLabel,
              semanticsLabel: maxLabel,
            ),
          ),
        ],
      ),
    );
  }
}

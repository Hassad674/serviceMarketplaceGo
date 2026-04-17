/// did_you_mean_banner.dart — rose-themed inline banner surfaced by
/// [SearchScreen] when the server returns a `corrected_query`.
/// Tapping the suggestion reruns the search with the corrected
/// string. Mirrors `web/src/shared/components/search/did-you-mean-banner.tsx`.
library;

import 'package:flutter/material.dart';

import 'filter_sections/filter_primitives.dart';

class DidYouMeanBanner extends StatelessWidget {
  const DidYouMeanBanner({
    super.key,
    required this.suggestion,
    required this.onApply,
    required this.label,
  });

  final String suggestion;
  final VoidCallback onApply;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      key: const ValueKey('did-you-mean-banner'),
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: kFilterRose100,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: kFilterRose500.withValues(alpha: 0.25),
        ),
      ),
      child: Row(
        children: [
          const Icon(
            Icons.lightbulb_outline,
            size: 18,
            color: kFilterRose700,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: RichText(
              text: TextSpan(
                style: const TextStyle(
                  color: kFilterRose700,
                  fontSize: 14,
                ),
                children: [
                  TextSpan(text: '$label '),
                  TextSpan(
                    text: '"$suggestion"',
                    style: const TextStyle(fontWeight: FontWeight.w700),
                  ),
                  const TextSpan(text: ' ?'),
                ],
              ),
            ),
          ),
          TextButton(
            onPressed: onApply,
            style: TextButton.styleFrom(foregroundColor: kFilterRose700),
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }
}

/// did_you_mean_banner.dart — Soleil v2 inline banner surfaced by
/// [SearchScreen] when the server returns a `corrected_query`.
/// Tapping the suggestion reruns the search with the corrected
/// string. Mirrors `web/src/shared/components/search/did-you-mean-banner.tsx`.
library;

import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
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
    final theme = Theme.of(context);
    final colorScheme = theme.colorScheme;
    final colors = theme.extension<AppColors>();
    final tint = colors?.accentSoft ?? kFilterRose100;
    final fg = colors?.primaryDeep ?? kFilterRose700;
    return Container(
      key: const ValueKey('did-you-mean-banner'),
      margin: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: tint,
        borderRadius: BorderRadius.circular(AppTheme.radiusFull),
        border: Border.all(
          color: colorScheme.primary.withValues(alpha: 0.25),
        ),
      ),
      child: Row(
        children: [
          Icon(
            Icons.lightbulb_outline_rounded,
            size: 18,
            color: fg,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: RichText(
              text: TextSpan(
                style: SoleilTextStyles.body.copyWith(
                  color: fg,
                  fontSize: 13.5,
                ),
                children: [
                  TextSpan(text: '$label '),
                  TextSpan(
                    text: '"$suggestion"',
                    style: const TextStyle(
                      fontWeight: FontWeight.w700,
                      fontStyle: FontStyle.italic,
                    ),
                  ),
                  const TextSpan(text: ' ?'),
                ],
              ),
            ),
          ),
          TextButton(
            onPressed: onApply,
            style: TextButton.styleFrom(foregroundColor: fg),
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }
}

import 'package:flutter/material.dart';

import '../../../../../core/theme/app_theme.dart';

/// Soleil editorial hero shown at the top of [CreateJobScreen].
///
/// Corail eyebrow + Fraunces display title with italic-corail accent
/// + tabac subtitle. Pure presentation — no state, no callbacks.
///
/// Extracted from `create_job_screen.dart` as part of the NF-9 file
/// split (V7 audit). Behaviour is unchanged.
class CreateJobSoleilHero extends StatelessWidget {
  const CreateJobSoleilHero({
    super.key,
    required this.eyebrow,
    required this.titlePrefix,
    required this.titleAccent,
    required this.subtitle,
  });

  final String eyebrow;
  final String titlePrefix;
  final String titleAccent;
  final String subtitle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final primary = theme.colorScheme.primary;
    final mute = appColors?.mutedForeground ?? theme.colorScheme.onSurfaceVariant;

    return Padding(
      padding: const EdgeInsets.only(top: 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            eyebrow,
            style: SoleilTextStyles.mono.copyWith(
              color: primary,
              fontSize: 11,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.4,
            ),
          ),
          const SizedBox(height: 8),
          RichText(
            text: TextSpan(
              style: SoleilTextStyles.displayM.copyWith(
                color: theme.colorScheme.onSurface,
              ),
              children: [
                TextSpan(text: '$titlePrefix '),
                TextSpan(
                  text: titleAccent,
                  style: SoleilTextStyles.displayM.copyWith(
                    fontStyle: FontStyle.italic,
                    color: primary,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 8),
          Text(
            subtitle,
            style: SoleilTextStyles.body.copyWith(color: mute, fontSize: 13.5),
          ),
        ],
      ),
    );
  }
}

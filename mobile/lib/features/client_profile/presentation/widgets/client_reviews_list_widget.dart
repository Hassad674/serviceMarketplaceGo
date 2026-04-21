import 'package:flutter/material.dart';

import '../../../../core/models/review.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/widgets/review_card_widget.dart';

/// Renders the list of reviews received by the client from providers.
///
/// Reuses the shared [ReviewCardWidget] so the visual style stays in
/// sync with the project-history reviews on provider profiles.
class ClientReviewsListWidget extends StatelessWidget {
  const ClientReviewsListWidget({
    super.key,
    required this.reviews,
  });

  final List<Review> reviews;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final appColors = theme.extension<AppColors>();

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        boxShadow: AppTheme.cardShadow,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.reviews_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.clientProfileReviewsTitle,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 12),
          if (reviews.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Text(
                l10n.clientProfileReviewsEmpty,
                style: theme.textTheme.bodyMedium?.copyWith(
                  color: appColors?.mutedForeground,
                  fontStyle: FontStyle.italic,
                ),
              ),
            )
          else
            // Short lists — render inline. Pagination is not in scope
            // for this feature and the backend already slices to the
            // most recent rows.
            for (final review in reviews) ...[
              ReviewCardWidget(review: review),
              const SizedBox(height: 8),
            ],
        ],
      ),
    );
  }
}

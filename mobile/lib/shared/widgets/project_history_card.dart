import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../../core/models/review.dart';
import 'review_card_widget.dart';

/// Shared card rendering one completed project — amount pill + date
/// header, optional title, embedded review (or "Awaiting review"
/// placeholder), and an optional [footer] slot consumers can use to
/// append context such as a counterparty chip.
///
/// Used both by the provider-side project history (under
/// `features/project_history/`) and by the client-profile project
/// history (under `features/client_profile/`) so the visual pattern
/// stays in sync across surfaces.
class ProjectHistoryCard extends StatelessWidget {
  const ProjectHistoryCard({
    super.key,
    required this.title,
    required this.amountCents,
    required this.completedAt,
    this.review,
    this.footer,
  });

  /// Proposal title. Renders only when non-empty (callers may hide
  /// the title when the other party opted out of sharing it).
  final String title;

  /// Amount in cents. Formatted as a grouped euro value for the pill.
  final int amountCents;

  /// Timestamp the proposal was marked completed.
  final DateTime completedAt;

  /// Optional review to embed inside the card. When null the card
  /// renders an "Awaiting review" placeholder.
  final Review? review;

  /// Optional widget appended after the review body. Used on the
  /// client surface to attach the provider chip.
  final Widget? footer;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final amountEuros = amountCents / 100;
    final formattedAmount = NumberFormat.currency(
      locale: 'fr_FR',
      symbol: '€',
      decimalDigits: 0,
    ).format(amountEuros);
    final formattedDate =
        DateFormat('d MMM yyyy', 'fr_FR').format(completedAt);

    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: theme.cardColor,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header: amount pill + date
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Container(
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 5,
                ),
                decoration: BoxDecoration(
                  gradient: const LinearGradient(
                    colors: [Color(0xFFFFE4E6), Color(0xFFFEF2F2)],
                  ),
                  borderRadius: BorderRadius.circular(99),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(
                      Icons.euro,
                      size: 13,
                      color: Color(0xFFBE123C),
                    ),
                    const SizedBox(width: 3),
                    Text(
                      formattedAmount.replaceAll('€', '').trim(),
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w700,
                        color: Color(0xFFBE123C),
                      ),
                    ),
                  ],
                ),
              ),
              Row(
                children: [
                  Icon(
                    Icons.schedule,
                    size: 13,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                  const SizedBox(width: 4),
                  Text(
                    formattedDate,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ],
          ),
          if (title.isNotEmpty) ...[
            const SizedBox(height: 10),
            Text(
              title,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
          const SizedBox(height: 12),

          // Body: review or awaiting state
          if (review != null)
            ReviewCardWidget(review: review!)
          else
            const AwaitingReviewBox(),

          if (footer != null) ...[
            const SizedBox(height: 10),
            footer!,
          ],
        ],
      ),
    );
  }
}

/// Placeholder rendered inside [ProjectHistoryCard] when no review has
/// been submitted yet. Exposed publicly so feature-level widgets can
/// reuse the same visual outside of a card body if ever needed.
class AwaitingReviewBox extends StatelessWidget {
  const AwaitingReviewBox({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerLow,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: theme.dividerColor,
          style: BorderStyle.solid,
        ),
      ),
      child: Row(
        children: [
          Icon(
            Icons.schedule,
            size: 16,
            color: theme.colorScheme.onSurfaceVariant,
          ),
          const SizedBox(width: 8),
          Text(
            'Awaiting review',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}

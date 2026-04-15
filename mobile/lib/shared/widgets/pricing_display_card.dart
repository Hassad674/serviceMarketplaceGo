import 'package:flutter/material.dart';

import 'profile_display_card_shell.dart';

/// Read-only pricing card used by the public freelance and referrer
/// profile screens. Pure display widget: accepts an already-formatted
/// amount label plus optional note and negotiable flag. Collapses to
/// `SizedBox.shrink()` when [amountLabel] is empty so public viewers
/// never see a placeholder card.
///
/// Lives under `shared/widgets/` so both profile features mount the
/// same card without a cross-feature import.
class PricingDisplayCard extends StatelessWidget {
  const PricingDisplayCard({
    super.key,
    required this.title,
    required this.amountLabel,
    required this.note,
    required this.negotiable,
    required this.negotiableBadgeLabel,
  });

  final String title;

  /// Already-formatted amount string (e.g. `500 € / j`). Empty
  /// string collapses the card.
  final String amountLabel;

  /// Optional free-form note rendered under the amount. Empty string
  /// hides the line entirely.
  final String note;

  /// Renders a pill next to the amount when true.
  final bool negotiable;

  /// Localized label used inside the negotiable pill.
  final String negotiableBadgeLabel;

  @override
  Widget build(BuildContext context) {
    if (amountLabel.isEmpty) return const SizedBox.shrink();
    final theme = Theme.of(context);
    return ProfileDisplayCardShell(
      title: title,
      icon: Icons.paid_outlined,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Wrap(
            crossAxisAlignment: WrapCrossAlignment.center,
            spacing: 8,
            runSpacing: 6,
            children: [
              Text(
                amountLabel,
                style: theme.textTheme.titleLarge?.copyWith(
                  fontWeight: FontWeight.w700,
                ),
              ),
              if (negotiable) _NegotiableBadge(label: negotiableBadgeLabel),
            ],
          ),
          if (note.isNotEmpty) ...[
            const SizedBox(height: 10),
            Text(
              note,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
                fontStyle: FontStyle.italic,
                height: 1.4,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _NegotiableBadge extends StatelessWidget {
  const _NegotiableBadge({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: Colors.green.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: Colors.green.withValues(alpha: 0.35)),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: Colors.green.shade800,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.2,
        ),
      ),
    );
  }
}

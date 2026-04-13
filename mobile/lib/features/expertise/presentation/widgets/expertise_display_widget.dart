import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/expertise_catalog.dart';
import '../utils/expertise_labels.dart';

/// Read-only public pill display of an organization's expertise.
///
/// Renders a card with a horizontal `Wrap` of rose-tinted pills
/// labelled with the localized domain names. The widget collapses
/// to a `SizedBox.shrink()` when there is nothing to show — on
/// public profiles we hide the whole section rather than display
/// an empty placeholder.
class ExpertiseDisplayWidget extends StatelessWidget {
  const ExpertiseDisplayWidget({
    super.key,
    required this.domains,
  });

  /// Server-provided list of domain keys. Unknown keys (possible
  /// during rolling deploys) are filtered out defensively so the
  /// widget never shows raw backend identifiers.
  final List<String> domains;

  @override
  Widget build(BuildContext context) {
    final visible = domains
        .where(ExpertiseCatalog.isKnownKey)
        .toList(growable: false);
    if (visible.isEmpty) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

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
                Icons.auto_awesome_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  l10n.expertiseSectionTitle,
                  style: theme.textTheme.titleMedium,
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final key in visible)
                _ExpertisePill(label: localizedExpertiseLabel(context, key)),
            ],
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Private pill — rose-tinted, read-only chip
// ---------------------------------------------------------------------------

class _ExpertisePill extends StatelessWidget {
  const _ExpertisePill({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;

    return Semantics(
      label: label,
      container: true,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: primary.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: primary.withValues(alpha: 0.2)),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: primary,
            fontWeight: FontWeight.w600,
            fontSize: 13,
          ),
        ),
      ),
    );
  }
}

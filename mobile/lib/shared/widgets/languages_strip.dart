import 'package:flutter/material.dart';

/// Generic read-only languages strip. Renders two horizontal
/// groups of language chips: professional (left) and conversational
/// (right), separated by a subtle label. Pure display widget.
///
/// Callers pass already-localized language labels. This widget does
/// not know about ISO codes, catalogs, or l10n — it just paints.
class LanguagesStrip extends StatelessWidget {
  const LanguagesStrip({
    super.key,
    required this.professional,
    required this.conversational,
    required this.professionalHeader,
    required this.conversationalHeader,
  });

  /// Already-localized labels (e.g. `["Français", "English"]`).
  final List<String> professional;

  /// Already-localized labels.
  final List<String> conversational;

  /// Localized header for the professional bucket.
  final String professionalHeader;

  /// Localized header for the conversational bucket.
  final String conversationalHeader;

  @override
  Widget build(BuildContext context) {
    if (professional.isEmpty && conversational.isEmpty) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (professional.isNotEmpty)
          _LanguageGroup(header: professionalHeader, labels: professional),
        if (professional.isNotEmpty && conversational.isNotEmpty)
          const SizedBox(height: 8),
        if (conversational.isNotEmpty)
          _LanguageGroup(header: conversationalHeader, labels: conversational),
      ],
    );
  }
}

class _LanguageGroup extends StatelessWidget {
  const _LanguageGroup({required this.header, required this.labels});

  final String header;
  final List<String> labels;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          header,
          style: theme.textTheme.labelSmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
            fontWeight: FontWeight.w600,
            letterSpacing: 0.3,
          ),
        ),
        const SizedBox(height: 6),
        Wrap(
          spacing: 6,
          runSpacing: 6,
          children: [
            for (final label in labels) _LanguageChip(label: label),
          ],
        ),
      ],
    );
  }
}

class _LanguageChip extends StatelessWidget {
  const _LanguageChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

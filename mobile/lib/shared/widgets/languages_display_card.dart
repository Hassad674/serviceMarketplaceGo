import 'package:flutter/material.dart';

import '../profile/flag_emoji.dart';
import '../profile/language_catalog.dart';
import 'profile_display_card_shell.dart';

/// Read-only languages card used by the public freelance and
/// referrer profile screens. Accepts ISO 639-1 codes and resolves
/// labels + flag emojis from the shared catalog, so both features
/// render identical chips.
///
/// Collapses to `SizedBox.shrink()` when both groups are empty.
class LanguagesDisplayCard extends StatelessWidget {
  const LanguagesDisplayCard({
    super.key,
    required this.title,
    required this.professional,
    required this.conversational,
    required this.professionalHeader,
    required this.conversationalHeader,
    required this.locale,
  });

  final String title;

  /// ISO 639-1 codes (e.g. `['fr', 'en']`). Empty list hides the
  /// group entirely.
  final List<String> professional;

  final List<String> conversational;

  /// Already-localized header for the professional bucket.
  final String professionalHeader;

  /// Already-localized header for the conversational bucket.
  final String conversationalHeader;

  /// Two-letter language code used to resolve catalog labels.
  final String locale;

  @override
  Widget build(BuildContext context) {
    if (professional.isEmpty && conversational.isEmpty) {
      return const SizedBox.shrink();
    }
    return ProfileDisplayCardShell(
      title: title,
      icon: Icons.translate_outlined,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (professional.isNotEmpty)
            _LanguageGroup(
              header: professionalHeader,
              codes: professional,
              locale: locale,
            ),
          if (professional.isNotEmpty && conversational.isNotEmpty)
            const SizedBox(height: 12),
          if (conversational.isNotEmpty)
            _LanguageGroup(
              header: conversationalHeader,
              codes: conversational,
              locale: locale,
            ),
        ],
      ),
    );
  }
}

class _LanguageGroup extends StatelessWidget {
  const _LanguageGroup({
    required this.header,
    required this.codes,
    required this.locale,
  });

  final String header;
  final List<String> codes;
  final String locale;

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
            for (final code in codes)
              _LanguageChip(
                flag: _flagFor(code),
                label: LanguageCatalog.labelFor(code, locale: locale),
              ),
          ],
        ),
      ],
    );
  }

  String _flagFor(String code) {
    final entry = LanguageCatalog.findByCode(code);
    if (entry == null) return '';
    return countryCodeToFlagEmoji(entry.flagCountryCode);
  }
}

class _LanguageChip extends StatelessWidget {
  const _LanguageChip({required this.flag, required this.label});

  final String flag;
  final String label;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (flag.isNotEmpty) ...[
            Text(flag, style: const TextStyle(fontSize: 13)),
            const SizedBox(width: 4),
          ],
          Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
        ],
      ),
    );
  }
}

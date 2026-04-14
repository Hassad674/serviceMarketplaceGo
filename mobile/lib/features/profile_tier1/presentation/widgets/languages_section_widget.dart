import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/languages.dart';
import '../providers/profile_tier1_providers.dart';
import '../utils/language_catalog.dart';
import 'language_picker_bottom_sheet.dart';

/// Compact "languages" card. Two chip rows: professional and
/// conversational. Tapping the edit button opens two modal bottom
/// sheets sequentially — first professional, then conversational —
/// to keep each sheet focused on a single bucket.
class LanguagesSectionWidget extends ConsumerStatefulWidget {
  const LanguagesSectionWidget({
    super.key,
    required this.initialLanguages,
    required this.canEdit,
    required this.onSaved,
  });

  final Languages initialLanguages;
  final bool canEdit;
  final VoidCallback onSaved;

  @override
  ConsumerState<LanguagesSectionWidget> createState() =>
      _LanguagesSectionWidgetState();
}

class _LanguagesSectionWidgetState
    extends ConsumerState<LanguagesSectionWidget> {
  late Languages _pending;

  @override
  void initState() {
    super.initState();
    _pending = widget.initialLanguages;
  }

  @override
  void didUpdateWidget(covariant LanguagesSectionWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initialLanguages != widget.initialLanguages) {
      _pending = widget.initialLanguages;
    }
  }

  @override
  Widget build(BuildContext context) {
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
                Icons.language_outlined,
                size: 20,
                color: theme.colorScheme.primary,
              ),
              const SizedBox(width: 8),
              Text(
                l10n.tier1LanguagesSectionTitle,
                style: theme.textTheme.titleMedium,
              ),
            ],
          ),
          const SizedBox(height: 12),
          if (_pending.isEmpty)
            _EmptyState(text: l10n.tier1LanguagesEmpty)
          else
            _LanguagesSummary(languages: _pending),
          if (widget.canEdit) ...[
            const SizedBox(height: 12),
            _EditButton(
              label: l10n.tier1LanguagesEditButton,
              onTap: _openEditor,
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _openEditor() async {
    final l10n = AppLocalizations.of(context)!;
    final pro = await showLanguagePickerBottomSheet(
      context: context,
      title: l10n.tier1LanguagesProfessionalLabel,
      initialCodes: _pending.professional,
    );
    if (pro == null) return;
    if (!mounted) return;

    final conv = await showLanguagePickerBottomSheet(
      context: context,
      title: l10n.tier1LanguagesConversationalLabel,
      initialCodes: _pending.conversational,
    );
    if (conv == null) return;
    if (!mounted) return;

    final previous = _pending;
    final next = Languages(professional: pro, conversational: conv);
    setState(() => _pending = next);

    final ok = await ref
        .read(languagesEditorProvider.notifier)
        .save(pro, conv);
    if (!mounted) return;
    if (!ok) {
      setState(() => _pending = previous);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(l10n.tier1ErrorGeneric),
          behavior: SnackBarBehavior.floating,
        ),
      );
      return;
    }
    widget.onSaved();
  }
}

// ---------------------------------------------------------------------------
// Read-only summary (also reused by the public identity strip indirectly)
// ---------------------------------------------------------------------------

class _LanguagesSummary extends StatelessWidget {
  const _LanguagesSummary({required this.languages});

  final Languages languages;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (languages.professional.isNotEmpty) ...[
          Text(
            l10n.tier1LanguagesProfessionalLabel,
            style: theme.textTheme.labelMedium,
          ),
          const SizedBox(height: 6),
          _LanguageChipRow(
            codes: languages.professional,
            locale: locale,
          ),
          const SizedBox(height: 12),
        ],
        if (languages.conversational.isNotEmpty) ...[
          Text(
            l10n.tier1LanguagesConversationalLabel,
            style: theme.textTheme.labelMedium,
          ),
          const SizedBox(height: 6),
          _LanguageChipRow(
            codes: languages.conversational,
            locale: locale,
          ),
        ],
      ],
    );
  }
}

class _LanguageChipRow extends StatelessWidget {
  const _LanguageChipRow({required this.codes, required this.locale});

  final List<String> codes;
  final String locale;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Wrap(
      spacing: 6,
      runSpacing: 6,
      children: [
        for (final code in codes)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
            decoration: BoxDecoration(
              color: theme.colorScheme.primary.withValues(alpha: 0.08),
              borderRadius: BorderRadius.circular(999),
              border: Border.all(
                color: theme.colorScheme.primary.withValues(alpha: 0.2),
              ),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.public,
                  size: 14,
                  color: theme.colorScheme.primary.withValues(alpha: 0.75),
                ),
                const SizedBox(width: 6),
                Text(
                  LanguageCatalog.labelFor(code, locale: locale),
                  style: theme.textTheme.labelMedium?.copyWith(
                    color: theme.colorScheme.primary,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(vertical: 16, horizontal: 14),
      decoration: BoxDecoration(
        color: appColors?.muted,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: appColors?.border ?? theme.dividerColor),
      ),
      child: Row(
        children: [
          Icon(
            Icons.info_outline,
            size: 18,
            color: appColors?.mutedForeground,
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              text,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: appColors?.mutedForeground,
                height: 1.4,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EditButton extends StatelessWidget {
  const _EditButton({required this.label, required this.onTap});

  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: double.infinity,
      child: OutlinedButton.icon(
        onPressed: onTap,
        icon: const Icon(Icons.edit_outlined, size: 18),
        label: Text(label),
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(double.infinity, 48),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
        ),
      ),
    );
  }
}

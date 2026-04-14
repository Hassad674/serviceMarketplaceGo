import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../../../shared/profile/language_catalog.dart';
import '../../../../shared/widgets/languages_strip.dart';
import '../../domain/entities/organization_shared_profile.dart';
import '../providers/organization_shared_providers.dart';

/// Editable languages card rendered on the freelance profile screen.
/// Reads + writes the shared org block via the organization_shared
/// feature.
class SharedLanguagesSectionWidget extends ConsumerStatefulWidget {
  const SharedLanguagesSectionWidget({
    super.key,
    required this.initial,
    required this.canEdit,
    required this.onSaved,
  });

  final OrganizationSharedProfile initial;
  final bool canEdit;
  final VoidCallback onSaved;

  @override
  ConsumerState<SharedLanguagesSectionWidget> createState() =>
      _SharedLanguagesSectionWidgetState();
}

class _SharedLanguagesSectionWidgetState
    extends ConsumerState<SharedLanguagesSectionWidget> {
  late List<String> _professional;
  late List<String> _conversational;

  @override
  void initState() {
    super.initState();
    _hydrate();
  }

  @override
  void didUpdateWidget(covariant SharedLanguagesSectionWidget oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.initial != widget.initial) {
      _hydrate();
    }
  }

  void _hydrate() {
    _professional = List<String>.from(widget.initial.languagesProfessional);
    _conversational =
        List<String>.from(widget.initial.languagesConversational);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final hasAny = _professional.isNotEmpty || _conversational.isNotEmpty;

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
                Icons.translate_outlined,
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
          if (hasAny)
            LanguagesStrip(
              professional: _professional
                  .map((code) => LanguageCatalog.labelFor(code, locale: locale))
                  .toList(),
              conversational: _conversational
                  .map((code) => LanguageCatalog.labelFor(code, locale: locale))
                  .toList(),
              professionalHeader: l10n.tier1LanguagesProfessionalLabel,
              conversationalHeader: l10n.tier1LanguagesConversationalLabel,
            )
          else
            _EmptyHint(text: l10n.tier1LanguagesEmpty),
          if (widget.canEdit) ...[
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: _openEditor,
              icon: const Icon(Icons.edit_outlined, size: 18),
              label: Text(l10n.tier1LanguagesEditButton),
              style: OutlinedButton.styleFrom(
                minimumSize: const Size(double.infinity, 48),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _openEditor() async {
    final result = await showModalBottomSheet<_LanguagesDraft>(
      context: context,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => _LanguagesEditorSheet(
        initialProfessional: _professional,
        initialConversational: _conversational,
      ),
    );
    if (result == null || !mounted) return;

    setState(() {
      _professional = result.professional;
      _conversational = result.conversational;
    });

    final ok = await ref.read(sharedLanguagesEditorProvider.notifier).save(
          professional: result.professional,
          conversational: result.conversational,
        );
    if (!mounted) return;
    if (!ok) {
      final l10n = AppLocalizations.of(context)!;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(l10n.tier1ErrorGeneric)),
      );
      return;
    }
    widget.onSaved();
  }
}

// ---------------------------------------------------------------------------
// Editor bottom sheet
// ---------------------------------------------------------------------------

class _LanguagesDraft {
  const _LanguagesDraft({
    required this.professional,
    required this.conversational,
  });

  final List<String> professional;
  final List<String> conversational;
}

class _LanguagesEditorSheet extends StatefulWidget {
  const _LanguagesEditorSheet({
    required this.initialProfessional,
    required this.initialConversational,
  });

  final List<String> initialProfessional;
  final List<String> initialConversational;

  @override
  State<_LanguagesEditorSheet> createState() => _LanguagesEditorSheetState();
}

class _LanguagesEditorSheetState extends State<_LanguagesEditorSheet> {
  late Set<String> _professional;
  late Set<String> _conversational;
  String _query = '';

  @override
  void initState() {
    super.initState();
    _professional = Set<String>.from(widget.initialProfessional);
    _conversational = Set<String>.from(widget.initialConversational);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final locale = Localizations.localeOf(context).languageCode;
    final isFrench = locale.startsWith('fr');
    final filtered = _filterEntries(isFrench);

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.75,
        child: Column(
          children: [
            const SizedBox(height: 12),
            Container(
              width: 40,
              height: 4,
              decoration: BoxDecoration(
                color: Theme.of(context).dividerColor,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(16),
              child: TextField(
                decoration: InputDecoration(
                  prefixIcon: const Icon(Icons.search, size: 18),
                  hintText: l10n.tier1LanguagesSearchPlaceholder,
                  border: const OutlineInputBorder(),
                ),
                onChanged: (value) => setState(() => _query = value),
              ),
            ),
            Expanded(
              child: ListView.builder(
                itemCount: filtered.length,
                itemBuilder: (context, index) {
                  final entry = filtered[index];
                  final label = isFrench ? entry.labelFr : entry.labelEn;
                  return _LanguageRow(
                    entryCode: entry.code,
                    label: label,
                    isProfessional: _professional.contains(entry.code),
                    isConversational: _conversational.contains(entry.code),
                    professionalLabel: l10n.tier1LanguagesProfessionalLabel,
                    conversationalLabel: l10n.tier1LanguagesConversationalLabel,
                    onTogglePro: (value) => _togglePro(entry.code, value),
                    onToggleConv: (value) => _toggleConv(entry.code, value),
                  );
                },
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () => Navigator.of(context).pop(),
                      child: Text(l10n.tier1Cancel),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: ElevatedButton(
                      onPressed: _submit,
                      child: Text(l10n.tier1Save),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  List<LanguageEntry> _filterEntries(bool isFrench) {
    if (_query.isEmpty) return LanguageCatalog.entries;
    final normalized = _query.toLowerCase().trim();
    return LanguageCatalog.entries.where((entry) {
      final label = isFrench ? entry.labelFr : entry.labelEn;
      return label.toLowerCase().contains(normalized);
    }).toList();
  }

  void _togglePro(String code, bool value) {
    setState(() {
      if (value) {
        _professional.add(code);
        _conversational.remove(code);
      } else {
        _professional.remove(code);
      }
    });
  }

  void _toggleConv(String code, bool value) {
    setState(() {
      if (value) {
        _conversational.add(code);
        _professional.remove(code);
      } else {
        _conversational.remove(code);
      }
    });
  }

  void _submit() {
    Navigator.of(context).pop(
      _LanguagesDraft(
        professional: _professional.toList(),
        conversational: _conversational.toList(),
      ),
    );
  }
}

class _LanguageRow extends StatelessWidget {
  const _LanguageRow({
    required this.entryCode,
    required this.label,
    required this.isProfessional,
    required this.isConversational,
    required this.professionalLabel,
    required this.conversationalLabel,
    required this.onTogglePro,
    required this.onToggleConv,
  });

  final String entryCode;
  final String label;
  final bool isProfessional;
  final bool isConversational;
  final String professionalLabel;
  final String conversationalLabel;
  final ValueChanged<bool> onTogglePro;
  final ValueChanged<bool> onToggleConv;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
      child: Row(
        children: [
          Expanded(
            child: Text(label, style: Theme.of(context).textTheme.bodyLarge),
          ),
          _MiniToggle(
            label: professionalLabel,
            selected: isProfessional,
            onChanged: onTogglePro,
          ),
          const SizedBox(width: 8),
          _MiniToggle(
            label: conversationalLabel,
            selected: isConversational,
            onChanged: onToggleConv,
          ),
        ],
      ),
    );
  }
}

class _MiniToggle extends StatelessWidget {
  const _MiniToggle({
    required this.label,
    required this.selected,
    required this.onChanged,
  });

  final String label;
  final bool selected;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return FilterChip(
      label: Text(label, style: const TextStyle(fontSize: 11)),
      selected: selected,
      onSelected: onChanged,
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
      visualDensity: VisualDensity.compact,
    );
  }
}

class _EmptyHint extends StatelessWidget {
  const _EmptyHint({required this.text});

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

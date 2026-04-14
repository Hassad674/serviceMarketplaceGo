import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../utils/flag_emoji.dart';
import '../utils/language_catalog.dart';

/// Opens the multi-select language picker as a modal bottom sheet.
/// Resolves to the final list of ISO codes, or `null` when the user
/// dismissed the sheet without confirming.
Future<List<String>?> showLanguagePickerBottomSheet({
  required BuildContext context,
  required String title,
  required List<String> initialCodes,
}) {
  return showModalBottomSheet<List<String>>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => _LanguagePickerSheet(
      title: title,
      initialCodes: initialCodes,
    ),
  );
}

class _LanguagePickerSheet extends StatefulWidget {
  const _LanguagePickerSheet({
    required this.title,
    required this.initialCodes,
  });

  final String title;
  final List<String> initialCodes;

  @override
  State<_LanguagePickerSheet> createState() => _LanguagePickerSheetState();
}

class _LanguagePickerSheetState extends State<_LanguagePickerSheet> {
  final _query = TextEditingController();
  late Set<String> _selected;
  late String _locale;

  @override
  void initState() {
    super.initState();
    _selected = widget.initialCodes.map((c) => c.toLowerCase()).toSet();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _locale = Localizations.localeOf(context).languageCode;
  }

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  List<LanguageEntry> _filter() {
    final q = _query.text.trim().toLowerCase();
    if (q.isEmpty) return LanguageCatalog.entries;
    return LanguageCatalog.entries.where((e) {
      return e.labelEn.toLowerCase().contains(q) ||
          e.labelFr.toLowerCase().contains(q) ||
          e.code.contains(q);
    }).toList(growable: false);
  }

  void _toggle(String code) {
    setState(() {
      final lower = code.toLowerCase();
      if (_selected.contains(lower)) {
        _selected.remove(lower);
      } else {
        _selected.add(lower);
      }
    });
  }

  List<String> _orderedSelection() {
    // Preserve catalog order so the saved list stays deterministic.
    final out = <String>[];
    for (final e in LanguageCatalog.entries) {
      if (_selected.contains(e.code)) out.add(e.code);
    }
    // Include any codes the catalog did not know about (stale data).
    for (final code in _selected) {
      if (!out.contains(code)) out.add(code);
    }
    return out;
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final entries = _filter();

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.9,
          minChildSize: 0.5,
          maxChildSize: 0.95,
          builder: (_, scrollController) {
            return Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 16, 12, 8),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          widget.title,
                          style: theme.textTheme.titleLarge,
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.close),
                        onPressed: () => Navigator.of(context).pop(),
                      ),
                    ],
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 0, 20, 8),
                  child: Text(
                    l10n.tier1LanguagesCountLabel(_selected.length),
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.primary,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 0, 20, 12),
                  child: TextField(
                    controller: _query,
                    onChanged: (_) => setState(() {}),
                    decoration: InputDecoration(
                      prefixIcon: const Icon(Icons.search, size: 20),
                      hintText: l10n.tier1LanguagesSearchPlaceholder,
                      border: OutlineInputBorder(
                        borderRadius:
                            BorderRadius.circular(AppTheme.radiusMd),
                      ),
                    ),
                  ),
                ),
                const Divider(height: 1),
                Expanded(
                  child: ListView.builder(
                    controller: scrollController,
                    itemCount: entries.length,
                    itemBuilder: (ctx, i) {
                      final entry = entries[i];
                      final selected = _selected.contains(entry.code);
                      final label = _locale.startsWith('fr')
                          ? entry.labelFr
                          : entry.labelEn;
                      return CheckboxListTile(
                        value: selected,
                        onChanged: (_) => _toggle(entry.code),
                        title: Row(
                          children: [
                            Text(
                              countryCodeToFlagEmoji(entry.flagCountryCode),
                              style: const TextStyle(fontSize: 20),
                            ),
                            const SizedBox(width: 10),
                            Expanded(child: Text(label)),
                          ],
                        ),
                        controlAffinity: ListTileControlAffinity.trailing,
                      );
                    },
                  ),
                ),
                _SaveBar(
                  onSave: () => Navigator.of(context).pop(_orderedSelection()),
                ),
              ],
            );
          },
        ),
      ),
    );
  }
}

class _SaveBar extends StatelessWidget {
  const _SaveBar({required this.onSave});

  final VoidCallback onSave;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(color: appColors?.border ?? theme.dividerColor),
        ),
      ),
      child: SizedBox(
        width: double.infinity,
        child: ElevatedButton(
          onPressed: onSave,
          style: ElevatedButton.styleFrom(
            minimumSize: const Size(double.infinity, 48),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
          ),
          child: Text(l10n.tier1Save),
        ),
      ),
    );
  }
}

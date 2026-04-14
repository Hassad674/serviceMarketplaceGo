import 'package:flutter/material.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
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
    builder: (_) =>
        _LanguagePickerSheet(title: title, initialCodes: initialCodes),
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

  List<_MatchEntry> _filter() {
    final raw = _query.text.trim().toLowerCase();
    final out = <_MatchEntry>[];
    for (final entry in LanguageCatalog.entries) {
      final label = _locale.startsWith('fr') ? entry.labelFr : entry.labelEn;
      if (raw.isEmpty) {
        out.add(_MatchEntry(entry: entry, label: label, start: -1, end: -1));
        continue;
      }
      final idx = label.toLowerCase().indexOf(raw);
      if (idx >= 0) {
        out.add(
          _MatchEntry(
            entry: entry,
            label: label,
            start: idx,
            end: idx + raw.length,
          ),
        );
        continue;
      }
      if (entry.labelEn.toLowerCase().contains(raw) ||
          entry.labelFr.toLowerCase().contains(raw) ||
          entry.code.contains(raw)) {
        out.add(_MatchEntry(entry: entry, label: label, start: -1, end: -1));
      }
    }
    return out;
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

  void _clearAll() {
    setState(() => _selected.clear());
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
    final l10n = AppLocalizations.of(context)!;
    final matches = _filter();

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
          builder: (_, scrollController) => Column(
            children: [
              _SheetHeader(
                title: widget.title,
                selectedCount: _selected.length,
                onClearAll: _selected.isEmpty ? null : _clearAll,
              ),
              _SearchField(
                controller: _query,
                hintText: l10n.tier1LanguagesSearchPlaceholder,
                onChanged: () => setState(() {}),
              ),
              const Divider(height: 1),
              Expanded(
                child: _MatchList(
                  matches: matches,
                  selected: _selected,
                  scrollController: scrollController,
                  onToggle: _toggle,
                  emptyLabel: l10n.tier1LanguagesNoResults,
                ),
              ),
              _SaveBar(
                onSave: () => Navigator.of(context).pop(_orderedSelection()),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _MatchEntry {
  const _MatchEntry({
    required this.entry,
    required this.label,
    required this.start,
    required this.end,
  });

  final LanguageEntry entry;
  final String label;
  final int start;
  final int end;
}

class _SheetHeader extends StatelessWidget {
  const _SheetHeader({
    required this.title,
    required this.selectedCount,
    required this.onClearAll,
  });

  final String title;
  final int selectedCount;
  final VoidCallback? onClearAll;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 12, 8),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: theme.textTheme.titleLarge),
                const SizedBox(height: 2),
                Row(
                  children: [
                    Text(
                      l10n.tier1LanguagesCountLabel(selectedCount),
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.primary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    if (onClearAll != null) ...[
                      const SizedBox(width: 12),
                      TextButton(
                        onPressed: onClearAll,
                        style: TextButton.styleFrom(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 8,
                            vertical: 4,
                          ),
                          minimumSize: Size.zero,
                          tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        ),
                        child: Text(l10n.tier1LanguagesClearAll),
                      ),
                    ],
                  ],
                ),
              ],
            ),
          ),
          IconButton(
            icon: const Icon(Icons.close),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ],
      ),
    );
  }
}

class _SearchField extends StatelessWidget {
  const _SearchField({
    required this.controller,
    required this.hintText,
    required this.onChanged,
  });

  final TextEditingController controller;
  final String hintText;
  final VoidCallback onChanged;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 12),
      child: TextField(
        controller: controller,
        onChanged: (_) => onChanged(),
        textInputAction: TextInputAction.search,
        decoration: InputDecoration(
          prefixIcon: const Icon(Icons.search, size: 20),
          hintText: hintText,
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          ),
        ),
      ),
    );
  }
}

class _MatchList extends StatelessWidget {
  const _MatchList({
    required this.matches,
    required this.selected,
    required this.scrollController,
    required this.onToggle,
    required this.emptyLabel,
  });

  final List<_MatchEntry> matches;
  final Set<String> selected;
  final ScrollController scrollController;
  final void Function(String) onToggle;
  final String emptyLabel;

  @override
  Widget build(BuildContext context) {
    if (matches.isEmpty) {
      final theme = Theme.of(context);
      final appColors = theme.extension<AppColors>();
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            emptyLabel,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: appColors?.mutedForeground,
            ),
          ),
        ),
      );
    }
    return ListView.builder(
      controller: scrollController,
      itemCount: matches.length,
      itemBuilder: (ctx, i) {
        final match = matches[i];
        final isSelected = selected.contains(match.entry.code);
        return CheckboxListTile(
          value: isSelected,
          onChanged: (_) => onToggle(match.entry.code),
          title: Row(
            children: [
              Icon(
                Icons.public,
                size: 18,
                color: Theme.of(ctx).colorScheme.outline,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: _HighlightedLabel(
                  label: match.label,
                  start: match.start,
                  end: match.end,
                ),
              ),
            ],
          ),
          controlAffinity: ListTileControlAffinity.trailing,
        );
      },
    );
  }
}

class _HighlightedLabel extends StatelessWidget {
  const _HighlightedLabel({
    required this.label,
    required this.start,
    required this.end,
  });

  final String label;
  final int start;
  final int end;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (start < 0 || end <= start) {
      return Text(label, style: theme.textTheme.bodyMedium);
    }
    final before = label.substring(0, start);
    final hit = label.substring(start, end);
    final after = label.substring(end);
    return RichText(
      overflow: TextOverflow.ellipsis,
      text: TextSpan(
        style: theme.textTheme.bodyMedium,
        children: [
          TextSpan(text: before),
          TextSpan(
            text: hit,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.primary,
              fontWeight: FontWeight.w700,
            ),
          ),
          TextSpan(text: after),
        ],
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

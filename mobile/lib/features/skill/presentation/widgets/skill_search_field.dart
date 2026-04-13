import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/catalog_entry.dart';
import '../providers/skill_autocomplete_provider.dart';

/// Text input + debounced suggestion list for the editor bottom sheet.
///
/// The parent passes [existingSelections] so we can grey out skills
/// the user already picked — tapping a greyed row is a no-op.
///
/// Three callbacks let the parent stay in control of the draft list:
/// - [onPick] is called when the user taps a suggestion.
/// - [onCreate] is called when the user taps the "Create …" row
///   shown at the bottom when no exact match exists.
class SkillSearchField extends ConsumerStatefulWidget {
  const SkillSearchField({
    super.key,
    required this.existingSelections,
    required this.onPick,
    required this.onCreate,
  });

  final Set<String> existingSelections;
  final ValueChanged<CatalogEntry> onPick;
  final ValueChanged<String> onCreate;

  @override
  ConsumerState<SkillSearchField> createState() => _SkillSearchFieldState();
}

class _SkillSearchFieldState extends ConsumerState<SkillSearchField> {
  final TextEditingController _controller = TextEditingController();
  final FocusNode _focusNode = FocusNode();

  @override
  void dispose() {
    _controller.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  void _onChanged(String value) {
    ref.read(skillAutocompleteProvider.notifier).query(value);
    setState(() {}); // rebuild to refresh the "Create…" row visibility
  }

  void _handlePick(CatalogEntry entry) {
    widget.onPick(entry);
    _reset();
  }

  void _handleCreate() {
    final raw = _controller.text.trim();
    if (raw.isEmpty) return;
    widget.onCreate(raw);
    _reset();
  }

  void _reset() {
    _controller.clear();
    ref.read(skillAutocompleteProvider.notifier).clear();
    _focusNode.unfocus();
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final suggestions = ref.watch(skillAutocompleteProvider);
    final query = _controller.text.trim();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        TextField(
          controller: _controller,
          focusNode: _focusNode,
          textInputAction: TextInputAction.search,
          decoration: InputDecoration(
            hintText: l10n.skillsSearchPlaceholder,
            prefixIcon: const Icon(Icons.search, size: 20),
            suffixIcon: query.isNotEmpty
                ? IconButton(
                    icon: const Icon(Icons.close, size: 18),
                    onPressed: _reset,
                    tooltip: l10n.skillsCancel,
                  )
                : null,
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(12),
            ),
            isDense: true,
          ),
          onChanged: _onChanged,
        ),
        if (query.isNotEmpty) ...[
          const SizedBox(height: 8),
          _SuggestionList(
            state: suggestions,
            query: query,
            existingSelections: widget.existingSelections,
            onPick: _handlePick,
            onCreate: _handleCreate,
          ),
        ],
      ],
    );
  }
}

class _SuggestionList extends StatelessWidget {
  const _SuggestionList({
    required this.state,
    required this.query,
    required this.existingSelections,
    required this.onPick,
    required this.onCreate,
  });

  final AsyncValue<List<CatalogEntry>> state;
  final String query;
  final Set<String> existingSelections;
  final ValueChanged<CatalogEntry> onPick;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    return state.when(
      loading: () => const Padding(
        padding: EdgeInsets.symmetric(vertical: 12),
        child: Center(child: CircularProgressIndicator.adaptive()),
      ),
      error: (_, __) => _SuggestionError(onCreate: onCreate, query: query),
      data: (items) => _SuggestionResults(
        items: items,
        query: query,
        existingSelections: existingSelections,
        onPick: onPick,
        onCreate: onCreate,
      ),
    );
  }
}

class _SuggestionResults extends StatelessWidget {
  const _SuggestionResults({
    required this.items,
    required this.query,
    required this.existingSelections,
    required this.onPick,
    required this.onCreate,
  });

  final List<CatalogEntry> items;
  final String query;
  final Set<String> existingSelections;
  final ValueChanged<CatalogEntry> onPick;
  final VoidCallback onCreate;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    final hasExactMatch = items.any(
      (e) => e.displayText.toLowerCase() == query.toLowerCase() ||
          e.skillText == query.toLowerCase(),
    );

    final rows = <Widget>[
      for (final entry in items)
        _SuggestionRow(
          entry: entry,
          alreadySelected:
              existingSelections.contains(entry.skillText.toLowerCase()),
          onTap: () => onPick(entry),
        ),
      if (!hasExactMatch && query.isNotEmpty)
        ListTile(
          leading: const Icon(Icons.add_circle_outline, size: 22),
          title: Text(
            l10n.skillsCreateNew(query),
            style: theme.textTheme.bodyMedium?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          onTap: onCreate,
        ),
    ];

    if (rows.isEmpty) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 16),
        child: Text(
          l10n.skillsEmpty,
          style: theme.textTheme.bodySmall?.copyWith(color: theme.hintColor),
          textAlign: TextAlign.center,
        ),
      );
    }

    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: theme.dividerColor),
      ),
      constraints: const BoxConstraints(maxHeight: 220),
      child: ListView.separated(
        shrinkWrap: true,
        padding: EdgeInsets.zero,
        itemCount: rows.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (_, i) => rows[i],
      ),
    );
  }
}

class _SuggestionRow extends StatelessWidget {
  const _SuggestionRow({
    required this.entry,
    required this.alreadySelected,
    required this.onTap,
  });

  final CatalogEntry entry;
  final bool alreadySelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    return ListTile(
      title: Text(
        entry.displayText,
        style: theme.textTheme.bodyMedium?.copyWith(
          color: alreadySelected ? theme.disabledColor : null,
          fontWeight: FontWeight.w500,
        ),
      ),
      subtitle: entry.usageCount > 0
          ? Text(
              l10n.skillsUsageCount(entry.usageCount),
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.hintColor,
              ),
            )
          : null,
      trailing: alreadySelected
          ? Icon(Icons.check, size: 18, color: theme.colorScheme.primary)
          : const Icon(Icons.add, size: 18),
      onTap: alreadySelected ? null : onTap,
      dense: true,
    );
  }
}

class _SuggestionError extends StatelessWidget {
  const _SuggestionError({required this.onCreate, required this.query});

  final VoidCallback onCreate;
  final String query;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    final theme = Theme.of(context);
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 12),
          child: Text(
            l10n.skillsErrorGeneric,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.error,
            ),
          ),
        ),
        if (query.isNotEmpty)
          ListTile(
            leading: const Icon(Icons.add_circle_outline, size: 22),
            title: Text(l10n.skillsCreateNew(query)),
            onTap: onCreate,
          ),
      ],
    );
  }
}

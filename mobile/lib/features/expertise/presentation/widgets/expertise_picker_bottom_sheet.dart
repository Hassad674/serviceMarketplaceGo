import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/expertise_catalog.dart';
import '../providers/expertise_provider.dart';
import '../utils/expertise_labels.dart';

/// Opens the expertise picker as a modal bottom sheet and returns
/// the new selection on success (or `null` when the user cancels
/// or the save fails).
///
/// The picker is optimistic: it closes immediately once the user
/// taps "Save" and delegates error handling to the caller through
/// a follow-up SnackBar. This keeps the editor feeling instant on
/// flaky connections while still surfacing server rejections.
Future<List<String>?> showExpertisePickerBottomSheet({
  required BuildContext context,
  required List<String> initialDomains,
  required int maxSelection,
}) {
  return showModalBottomSheet<List<String>>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (ctx) => ExpertisePickerBottomSheet(
      initialDomains: initialDomains,
      maxSelection: maxSelection,
    ),
  );
}

// ---------------------------------------------------------------------------
// Picker widget
// ---------------------------------------------------------------------------

/// Bottom sheet that lets an operator pick up to [maxSelection]
/// expertise domains for their organization.
class ExpertisePickerBottomSheet extends ConsumerStatefulWidget {
  const ExpertisePickerBottomSheet({
    super.key,
    required this.initialDomains,
    required this.maxSelection,
  });

  final List<String> initialDomains;
  final int maxSelection;

  @override
  ConsumerState<ExpertisePickerBottomSheet> createState() =>
      _ExpertisePickerBottomSheetState();
}

class _ExpertisePickerBottomSheetState
    extends ConsumerState<ExpertisePickerBottomSheet> {
  late Set<String> _selected;

  @override
  void initState() {
    super.initState();
    // LinkedHashSet preserves insertion order so the "selected
    // pills" row on the private section stays stable.
    _selected = <String>{
      ...widget.initialDomains.where(ExpertiseCatalog.isKnownKey),
    };
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final editorState = ref.watch(expertiseEditorProvider);
    final maxReached = _selected.length >= widget.maxSelection;
    final hasChanges = !_sameSelection(_selected, widget.initialDomains);

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.75,
          minChildSize: 0.4,
          maxChildSize: 0.95,
          builder: (sheetCtx, scrollController) {
            return Column(
              children: [
                _Header(
                  count: _selected.length,
                  max: widget.maxSelection,
                ),
                const Divider(height: 1),
                if (maxReached)
                  _MaxReachedBanner(max: widget.maxSelection),
                Expanded(
                  child: ListView.builder(
                    controller: scrollController,
                    padding: EdgeInsets.zero,
                    itemCount: ExpertiseCatalog.allKeys.length,
                    itemBuilder: (listCtx, index) {
                      final key = ExpertiseCatalog.allKeys[index];
                      final isSelected = _selected.contains(key);
                      final isDisabled =
                          !isSelected && maxReached;
                      return _DomainRow(
                        label: localizedExpertiseLabel(listCtx, key),
                        selected: isSelected,
                        disabled: isDisabled,
                        onChanged: (_) => _toggle(key),
                      );
                    },
                  ),
                ),
                Container(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.surface,
                    border: Border(
                      top: BorderSide(
                        color: appColors?.border ?? theme.dividerColor,
                      ),
                    ),
                  ),
                  child: SizedBox(
                    width: double.infinity,
                    child: ElevatedButton(
                      onPressed: (!hasChanges || editorState.isSaving)
                          ? null
                          : _onSave,
                      style: ElevatedButton.styleFrom(
                        minimumSize: const Size(double.infinity, 48),
                        shape: RoundedRectangleBorder(
                          borderRadius:
                              BorderRadius.circular(AppTheme.radiusMd),
                        ),
                      ),
                      child: Text(
                        editorState.isSaving
                            ? l10n.expertiseSaving
                            : l10n.expertiseSave,
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Interaction handlers
  // --------------------------------------------------------------------------

  void _toggle(String key) {
    setState(() {
      if (_selected.contains(key)) {
        _selected.remove(key);
        return;
      }
      if (_selected.length >= widget.maxSelection) return;
      _selected.add(key);
    });
  }

  void _onSave() {
    // Preserve catalog order so agencies / freelancers can't reorder
    // the pills by tapping in a funny sequence — the private section
    // and public display both follow [ExpertiseCatalog.allKeys].
    final ordered = [
      for (final key in ExpertiseCatalog.allKeys)
        if (_selected.contains(key)) key,
    ];
    Navigator.of(context).pop(ordered);
  }

  // --------------------------------------------------------------------------
  // Pure helpers
  // --------------------------------------------------------------------------

  bool _sameSelection(Set<String> selected, List<String> initial) {
    final initialSet = initial.toSet();
    if (initialSet.length != selected.length) return false;
    return initialSet.containsAll(selected);
  }
}

// ---------------------------------------------------------------------------
// Header — title + counter
// ---------------------------------------------------------------------------

class _Header extends StatelessWidget {
  const _Header({required this.count, required this.max});

  final int count;
  final int max;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;

    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 12, 12),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  l10n.expertiseSectionTitle,
                  style: theme.textTheme.titleLarge,
                ),
                const SizedBox(height: 4),
                Text(
                  l10n.expertiseSectionSubtitle(max),
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: appColors?.mutedForeground,
                  ),
                ),
                const SizedBox(height: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: theme.colorScheme.primary.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    l10n.expertiseCounter(count, max),
                    style: TextStyle(
                      color: theme.colorScheme.primary,
                      fontWeight: FontWeight.w600,
                      fontSize: 12,
                    ),
                  ),
                ),
              ],
            ),
          ),
          IconButton(
            tooltip: MaterialLocalizations.of(context).closeButtonTooltip,
            icon: const Icon(Icons.close),
            onPressed: () => Navigator.of(context).pop(),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Max reached banner — subtle info strip
// ---------------------------------------------------------------------------

class _MaxReachedBanner extends StatelessWidget {
  const _MaxReachedBanner({required this.max});

  final int max;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final appColors = theme.extension<AppColors>();
    final l10n = AppLocalizations.of(context)!;
    final warn = appColors?.warning ?? theme.colorScheme.primary;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      color: warn.withValues(alpha: 0.08),
      child: Row(
        children: [
          Icon(Icons.info_outline, size: 16, color: warn),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              l10n.expertiseMaxReached(max),
              style: theme.textTheme.bodySmall?.copyWith(color: warn),
            ),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Domain row — accessible checkbox list tile
// ---------------------------------------------------------------------------

class _DomainRow extends StatelessWidget {
  const _DomainRow({
    required this.label,
    required this.selected,
    required this.disabled,
    required this.onChanged,
  });

  final String label;
  final bool selected;
  final bool disabled;
  final ValueChanged<bool?> onChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return CheckboxListTile(
      value: selected,
      onChanged: disabled ? null : onChanged,
      controlAffinity: ListTileControlAffinity.leading,
      activeColor: theme.colorScheme.primary,
      title: Text(
        label,
        style: theme.textTheme.bodyLarge?.copyWith(
          color: disabled
              ? theme.disabledColor
              : theme.textTheme.bodyLarge?.color,
        ),
      ),
      dense: false,
      // Keep the full tile tappable for a comfortable 56dp target.
      contentPadding: const EdgeInsets.symmetric(horizontal: 20),
    );
  }
}

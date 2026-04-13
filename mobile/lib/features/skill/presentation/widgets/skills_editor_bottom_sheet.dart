import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../l10n/app_localizations.dart';
import '../../domain/entities/catalog_entry.dart';
import '../../domain/entities/profile_skill.dart';
import '../../domain/entities/skill_limits.dart';
import '../providers/profile_skills_provider.dart';
import '../providers/skill_repository_provider.dart';
import 'expertise_skills_panel.dart';
import 'popular_skills_section.dart';
import 'skill_chip_widget.dart';
import 'skill_search_field.dart';

/// Opens the full skill editor as a modal bottom sheet and resolves
/// to `true` when the user successfully saved. The profile screen
/// uses the return value to invalidate the profile provider.
Future<bool> showSkillsEditorBottomSheet({
  required BuildContext context,
  required String? orgType,
  required List<String> expertiseKeys,
  required List<ProfileSkill> initialSkills,
}) async {
  final result = await showModalBottomSheet<bool>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    backgroundColor: Theme.of(context).colorScheme.surface,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (_) => SkillsEditorBottomSheet(
      orgType: orgType,
      expertiseKeys: expertiseKeys,
      initialSkills: initialSkills,
    ),
  );
  return result ?? false;
}

/// Draft selection editor. Keeps a local list of [_DraftSkill] so
/// users can add, remove, and reorder without hitting the backend
/// until they tap Save. On save, the full list is replaced atomically
/// via [ProfileSkillsNotifier.save].
class SkillsEditorBottomSheet extends ConsumerStatefulWidget {
  const SkillsEditorBottomSheet({
    super.key,
    required this.orgType,
    required this.expertiseKeys,
    required this.initialSkills,
  });

  final String? orgType;
  final List<String> expertiseKeys;
  final List<ProfileSkill> initialSkills;

  @override
  ConsumerState<SkillsEditorBottomSheet> createState() =>
      _SkillsEditorBottomSheetState();
}

/// One row in the draft selection. Stores display text so newly
/// created skills can render before the server round-trip completes.
class _DraftSkill {
  _DraftSkill({required this.skillText, required this.displayText});
  final String skillText;
  final String displayText;
}

class _SkillsEditorBottomSheetState
    extends ConsumerState<SkillsEditorBottomSheet> {
  late List<_DraftSkill> _draft;

  @override
  void initState() {
    super.initState();
    _draft = [
      for (final s in widget.initialSkills)
        _DraftSkill(
          skillText: s.skillText.toLowerCase(),
          displayText: s.displayText,
        ),
    ];
  }

  int get _max => SkillLimits.maxForOrgType(widget.orgType);
  bool get _maxReached => _draft.length >= _max;
  Set<String> get _selectedKeys =>
      {for (final d in _draft) d.skillText.toLowerCase()};

  bool get _hasChanges {
    if (_draft.length != widget.initialSkills.length) return true;
    for (var i = 0; i < _draft.length; i++) {
      if (_draft[i].skillText != widget.initialSkills[i].skillText) {
        return true;
      }
    }
    return false;
  }

  // --------------------------------------------------------------------------
  // Draft mutation
  // --------------------------------------------------------------------------

  void _addFromCatalog(CatalogEntry entry) {
    final key = entry.skillText.toLowerCase();
    if (key.isEmpty) return;
    if (_selectedKeys.contains(key)) return;
    if (_maxReached) {
      _showMaxReached();
      return;
    }
    setState(() {
      _draft.add(_DraftSkill(skillText: key, displayText: entry.displayText));
    });
  }

  void _remove(String skillText) {
    setState(() {
      _draft.removeWhere((d) => d.skillText == skillText);
    });
  }

  Future<void> _createFromQuery(String rawText) async {
    final trimmed = rawText.trim();
    if (trimmed.isEmpty) return;
    if (_maxReached) {
      _showMaxReached();
      return;
    }
    final key = trimmed.toLowerCase();
    if (_selectedKeys.contains(key)) return;

    // Optimistic insert — the backend will canonicalize on save.
    // If the POST fails, we surface an error and roll back.
    setState(() {
      _draft.add(_DraftSkill(skillText: key, displayText: trimmed));
    });

    try {
      final repo = ref.read(skillRepositoryProvider);
      final created = await repo.createUserSkill(trimmed);
      if (!mounted) return;
      setState(() {
        final idx = _draft.indexWhere((d) => d.skillText == key);
        if (idx >= 0) {
          _draft[idx] = _DraftSkill(
            skillText: created.skillText.toLowerCase(),
            displayText: created.displayText,
          );
        }
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _draft.removeWhere((d) => d.skillText == key);
      });
      _showError();
    }
  }

  // --------------------------------------------------------------------------
  // Save
  // --------------------------------------------------------------------------

  Future<void> _onSave() async {
    final notifier = ref.read(profileSkillsProvider.notifier);
    final ok = await notifier.save(
      [for (final d in _draft) d.skillText],
    );
    if (!mounted) return;
    if (ok) {
      Navigator.of(context).pop(true);
      return;
    }
    _showError();
  }

  // --------------------------------------------------------------------------
  // Snackbars
  // --------------------------------------------------------------------------

  void _showMaxReached() {
    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(l10n.skillsErrorTooMany(_max)),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  void _showError() {
    final l10n = AppLocalizations.of(context)!;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(l10n.skillsErrorGeneric),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  // --------------------------------------------------------------------------
  // Build
  // --------------------------------------------------------------------------

  @override
  Widget build(BuildContext context) {
    final editorState = ref.watch(profileSkillsProvider);

    return SafeArea(
      top: false,
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.85,
          minChildSize: 0.5,
          maxChildSize: 0.95,
          builder: (sheetCtx, scrollController) {
            return Column(
              children: [
                _EditorHeader(count: _draft.length, max: _max),
                const Divider(height: 1),
                Expanded(
                  child: ListView(
                    controller: scrollController,
                    padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
                    children: _buildBody(context),
                  ),
                ),
                _SaveBar(
                  hasChanges: _hasChanges,
                  isSaving: editorState.isSaving,
                  onSave: _onSave,
                ),
              ],
            );
          },
        ),
      ),
    );
  }

  List<Widget> _buildBody(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;
    // The "Popular in your domains" row stays contextual to the user's
    // declared expertise when they have any — it reads as curated
    // guidance. If no domains are declared we fall back to a sensible
    // default so the row always has content to surface.
    final popularKeys = widget.expertiseKeys.isNotEmpty
        ? widget.expertiseKeys
        : const <String>['development', 'design_ui_ux'];

    // The "Browse by domain" section always lists every expertise
    // domain (mirrors the web modal) so users can pick skills from
    // any area regardless of what they personally declared.
    const allKeys = SkillLimits.allExpertiseDomainKeys;

    return <Widget>[
      SkillSearchField(
        existingSelections: _selectedKeys,
        onPick: _addFromCatalog,
        onCreate: _createFromQuery,
      ),
      const SizedBox(height: 20),
      _SelectedChipsWrap(
        draft: _draft,
        onRemove: _remove,
      ),
      const SizedBox(height: 24),
      PopularSkillsSection(
        expertiseKeys: popularKeys,
        selectedKeys: _selectedKeys,
        onPick: _addFromCatalog,
      ),
      const SizedBox(height: 24),
      Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4),
        child: Text(
          l10n.skillsBrowseHeading,
          style: Theme.of(context).textTheme.titleSmall,
        ),
      ),
      const SizedBox(height: 12),
      for (final key in allKeys) ...[
        ExpertiseSkillsPanel(
          expertiseKey: key,
          selectedKeys: _selectedKeys,
          onPick: _addFromCatalog,
        ),
        const SizedBox(height: 8),
      ],
    ];
  }
}

// ---------------------------------------------------------------------------
// Header — title, subtitle, counter, close
// ---------------------------------------------------------------------------

class _EditorHeader extends StatelessWidget {
  const _EditorHeader({required this.count, required this.max});

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
                  l10n.skillsModalTitle,
                  style: theme.textTheme.titleLarge,
                ),
                const SizedBox(height: 4),
                Text(
                  l10n.skillsSectionSubtitle(max),
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
                    l10n.skillsCounter(count, max),
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
            onPressed: () => Navigator.of(context).pop(false),
          ),
        ],
      ),
    );
  }
}

// ---------------------------------------------------------------------------
// Selected chips — Wrap of deletable chips
// ---------------------------------------------------------------------------

class _SelectedChipsWrap extends StatelessWidget {
  const _SelectedChipsWrap({required this.draft, required this.onRemove});

  final List<_DraftSkill> draft;
  final ValueChanged<String> onRemove;

  @override
  Widget build(BuildContext context) {
    if (draft.isEmpty) return const SizedBox.shrink();
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (final d in draft)
          SkillChipWidget(
            label: d.displayText,
            onDeleted: () => onRemove(d.skillText),
          ),
      ],
    );
  }
}

// ---------------------------------------------------------------------------
// Save bar — persistent bottom CTA
// ---------------------------------------------------------------------------

class _SaveBar extends StatelessWidget {
  const _SaveBar({
    required this.hasChanges,
    required this.isSaving,
    required this.onSave,
  });

  final bool hasChanges;
  final bool isSaving;
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
          onPressed: (!hasChanges || isSaving) ? null : onSave,
          style: ElevatedButton.styleFrom(
            minimumSize: const Size(double.infinity, 48),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppTheme.radiusMd),
            ),
          ),
          child: Text(isSaving ? l10n.skillsSaving : l10n.skillsSave),
        ),
      ),
    );
  }
}

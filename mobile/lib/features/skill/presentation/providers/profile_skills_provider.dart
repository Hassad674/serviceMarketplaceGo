import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/profile_skill.dart';
import '../../domain/repositories/skill_repository.dart';
import 'skill_repository_provider.dart';

/// Immutable snapshot of the profile-skills editor state.
///
/// Holds three things:
/// - `skills`  — the current list of the operator's skills as an
///               [AsyncValue] so widgets can pattern-match loading,
///               error, and data states uniformly.
/// - `isSaving` — true while a `PUT /api/v1/profile/skills` is
///               in-flight so the save button can disable itself
///               without triggering a full skills reload.
/// - `error`   — last user-friendly error code (not a translated
///               string — the widget layer looks up the localized
///               copy). Cleared on the next successful op.
@immutable
class ProfileSkillsState {
  const ProfileSkillsState({
    required this.skills,
    this.isSaving = false,
    this.error,
  });

  final AsyncValue<List<ProfileSkill>> skills;
  final bool isSaving;
  final String? error;

  ProfileSkillsState copyWith({
    AsyncValue<List<ProfileSkill>>? skills,
    bool? isSaving,
    String? error,
    bool clearError = false,
  }) {
    return ProfileSkillsState(
      skills: skills ?? this.skills,
      isSaving: isSaving ?? this.isSaving,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

/// StateNotifier that owns the current operator's skill list.
///
/// Responsibilities:
/// - Load the initial list from the backend on construction.
/// - Expose an imperative `save(...)` method used by the editor.
/// - Expose `refresh()` for pull-to-refresh flows.
/// - Track an in-flight `isSaving` flag for the UI.
///
/// The notifier deliberately never mutates `skills` optimistically
/// on save — the editor manages its own draft list while the sheet
/// is open. On a successful save, we re-fetch so the source of
/// truth stays the server.
class ProfileSkillsNotifier extends StateNotifier<ProfileSkillsState> {
  ProfileSkillsNotifier(this._repository)
      : super(const ProfileSkillsState(skills: AsyncValue.loading())) {
    _load();
  }

  final SkillRepository _repository;

  /// Loads the initial list. Public variant used by `refresh()`.
  Future<void> _load() async {
    try {
      final list = await _repository.getProfileSkills();
      if (!mounted) return;
      state = state.copyWith(
        skills: AsyncValue.data(list),
        clearError: true,
      );
    } catch (error, stackTrace) {
      if (!mounted) return;
      debugPrint('ProfileSkillsNotifier._load failed: $error');
      state = state.copyWith(
        skills: AsyncValue.error(error, stackTrace),
      );
    }
  }

  /// Force a reload — used after navigating back from the editor
  /// or from pull-to-refresh on the profile screen.
  Future<void> refresh() => _load();

  /// Persists a new ordered list of skill texts. Returns `true` on
  /// success, `false` when the server rejected the payload. The
  /// caller is responsible for closing the editor sheet and showing
  /// a SnackBar on failure.
  Future<bool> save(List<String> skillTexts) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.updateProfileSkills(skillTexts);
      // Refetch so `skills` reflects server ordering + canonicalized
      // display_text values (the server lowercases keys and may
      // resolve user input to existing catalog rows).
      final fresh = await _repository.getProfileSkills();
      if (!mounted) return true;
      state = state.copyWith(
        skills: AsyncValue.data(fresh),
        isSaving: false,
        clearError: true,
      );
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('ProfileSkillsNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(
        isSaving: false,
        error: _mapError(error),
      );
      return false;
    }
  }

  /// Clears any error flag — called when the user re-opens the
  /// editor so stale toasts do not pop up again.
  void clearError() {
    if (state.error == null) return;
    state = state.copyWith(clearError: true);
  }

  /// Coarse error mapping. The widget layer converts the sentinel
  /// to a localized message via [AppLocalizations].
  String _mapError(Object error) => 'generic';
}

/// Riverpod wiring — presentation reads `ref.watch(profileSkillsProvider)`.
final profileSkillsProvider =
    StateNotifierProvider<ProfileSkillsNotifier, ProfileSkillsState>((ref) {
  final repo = ref.watch(skillRepositoryProvider);
  return ProfileSkillsNotifier(repo);
});

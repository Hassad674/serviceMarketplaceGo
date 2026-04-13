import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/expertise_repository_impl.dart';
import '../../domain/repositories/expertise_repository.dart';

// ---------------------------------------------------------------------------
// Dependency wiring
// ---------------------------------------------------------------------------

/// Injects the concrete [ExpertiseRepository] into the presentation
/// layer. Presentation code never imports from `data/` directly —
/// it depends on the abstract repository interface that lives in
/// `domain/`, which is the SOLID Dependency Inversion rule in
/// Riverpod form.
final expertiseRepositoryProvider = Provider<ExpertiseRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ExpertiseRepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Editor state
// ---------------------------------------------------------------------------

/// Immutable snapshot of the expertise editor's state.
@immutable
class ExpertiseEditorState {
  const ExpertiseEditorState({
    this.isSaving = false,
    this.error,
  });

  /// True while a save request is in-flight.
  final bool isSaving;

  /// User-friendly error message surfaced to the UI on the last
  /// failed save. `null` when there's nothing to report.
  final String? error;

  ExpertiseEditorState copyWith({
    bool? isSaving,
    String? error,
    bool clearError = false,
  }) {
    return ExpertiseEditorState(
      isSaving: isSaving ?? this.isSaving,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

// ---------------------------------------------------------------------------
// Editor notifier
// ---------------------------------------------------------------------------

/// Thin controller around [ExpertiseRepository] that exposes a
/// loading flag and a last-error for the picker bottom sheet.
///
/// It deliberately does NOT hold the "current" list of domains —
/// that lives on the profile provider and is fed back into the
/// widget via constructor props. This keeps the source of truth
/// single and avoids stale reads between the editor sheet and the
/// profile screen.
class ExpertiseEditorNotifier extends StateNotifier<ExpertiseEditorState> {
  ExpertiseEditorNotifier(this._repository)
      : super(const ExpertiseEditorState());

  final ExpertiseRepository _repository;

  /// Attempts to persist [domains]. Returns the server-echoed list
  /// on success, or `null` when the write failed. The caller is
  /// responsible for surfacing the error message — typically via a
  /// SnackBar after rolling back its optimistic state.
  Future<List<String>?> save(List<String> domains) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      final saved = await _repository.updateExpertise(domains);
      state = const ExpertiseEditorState();
      return saved;
    } catch (error, stackTrace) {
      debugPrint('ExpertiseEditorNotifier.save failed: $error\n$stackTrace');
      state = ExpertiseEditorState(
        isSaving: false,
        error: _mapError(error),
      );
      return null;
    }
  }

  /// Clears any error on the editor state (e.g. when the user
  /// re-opens the bottom sheet after a failed save).
  void clearError() {
    if (state.error == null) return;
    state = state.copyWith(clearError: true);
  }

  /// Maps raw exceptions to a stable, user-friendly sentinel. The
  /// UI looks up the localized message via
  /// [AppLocalizations.expertiseErrorGeneric] — keeping the mapping
  /// here means the widgets stay free of exception-type imports.
  String _mapError(Object error) => 'generic';
}

/// State notifier provider for the editor. Auto-disposes when the
/// picker sheet is closed, so the error flag resets between
/// sessions.
final expertiseEditorProvider = StateNotifierProvider.autoDispose<
    ExpertiseEditorNotifier, ExpertiseEditorState>((ref) {
  final repo = ref.watch(expertiseRepositoryProvider);
  return ExpertiseEditorNotifier(repo);
});

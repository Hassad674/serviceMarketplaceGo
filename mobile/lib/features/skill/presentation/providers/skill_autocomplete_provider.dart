import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/catalog_entry.dart';
import '../../domain/repositories/skill_repository.dart';
import 'skill_repository_provider.dart';

/// Debounced autocomplete StateNotifier for the skill search field.
///
/// Usage: the editor calls `query(value)` on every keystroke. The
/// notifier cancels any in-flight timer, waits 250ms, then hits
/// `GET /api/v1/skills/autocomplete`. An empty query short-circuits
/// to an empty data list so the suggestion panel hides immediately.
///
/// The state is an `AsyncValue<List<CatalogEntry>>` so the widget
/// can use the standard `.when(data, loading, error)` pattern.
class SkillAutocompleteNotifier
    extends StateNotifier<AsyncValue<List<CatalogEntry>>> {
  SkillAutocompleteNotifier(this._repository)
      : super(const AsyncValue<List<CatalogEntry>>.data(<CatalogEntry>[]));

  final SkillRepository _repository;
  Timer? _debounce;
  String _lastQuery = '';

  /// Enqueue a new search. Cancels any pending timer and any
  /// in-flight request is simply ignored on completion because we
  /// check `_lastQuery` before committing results.
  void query(String raw) {
    final trimmed = raw.trim();
    _debounce?.cancel();
    _lastQuery = trimmed;

    if (trimmed.isEmpty) {
      state = const AsyncValue<List<CatalogEntry>>.data(<CatalogEntry>[]);
      return;
    }

    state = const AsyncValue<List<CatalogEntry>>.loading();
    _debounce = Timer(const Duration(milliseconds: 250), () => _run(trimmed));
  }

  /// Clears the current state without triggering a new search —
  /// used when the editor bottom sheet closes.
  void clear() {
    _debounce?.cancel();
    _lastQuery = '';
    state = const AsyncValue<List<CatalogEntry>>.data(<CatalogEntry>[]);
  }

  Future<void> _run(String q) async {
    try {
      final results = await _repository.searchAutocomplete(q);
      if (!mounted) return;
      // Drop stale responses: the user has already typed something
      // else or cleared the field. The latest keystroke wins.
      if (_lastQuery != q) return;
      state = AsyncValue<List<CatalogEntry>>.data(results);
    } catch (error, stackTrace) {
      if (!mounted) return;
      if (_lastQuery != q) return;
      debugPrint('SkillAutocompleteNotifier._run failed: $error');
      state = AsyncValue<List<CatalogEntry>>.error(error, stackTrace);
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    super.dispose();
  }
}

/// Auto-dispose so each open of the editor bottom sheet starts
/// with a clean slate — no stale suggestions from a previous
/// session.
final skillAutocompleteProvider = StateNotifierProvider.autoDispose<
    SkillAutocompleteNotifier,
    AsyncValue<List<CatalogEntry>>>((ref) {
  final repo = ref.watch(skillRepositoryProvider);
  return SkillAutocompleteNotifier(repo);
});

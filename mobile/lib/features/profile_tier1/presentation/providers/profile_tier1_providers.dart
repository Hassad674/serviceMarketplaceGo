import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/profile_tier1_repository_impl.dart';
import '../../domain/entities/availability_status.dart';
import '../../domain/entities/location.dart';
import '../../domain/repositories/profile_tier1_repository.dart';

// ---------------------------------------------------------------------------
// Dependency wiring
// ---------------------------------------------------------------------------

/// Swappable DI seam for the Tier 1 feature. Tests override this
/// provider with an in-memory fake.
final profileTier1RepositoryProvider = Provider<ProfileTier1Repository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ProfileTier1RepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Editor state shared by all three PUT notifiers
// ---------------------------------------------------------------------------

/// Immutable snapshot used by every Tier 1 editor notifier.
///
/// The pricing sub-feature has its own list state so it is not
/// represented here; [profile_tier1_providers] only covers the
/// three simple PUT flows (location / languages / availability).
@immutable
class Tier1EditorState {
  const Tier1EditorState({
    this.isSaving = false,
    this.error,
  });

  final bool isSaving;
  final String? error;

  Tier1EditorState copyWith({
    bool? isSaving,
    String? error,
    bool clearError = false,
  }) {
    return Tier1EditorState(
      isSaving: isSaving ?? this.isSaving,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

// ---------------------------------------------------------------------------
// Location notifier
// ---------------------------------------------------------------------------

class LocationEditorNotifier extends StateNotifier<Tier1EditorState> {
  LocationEditorNotifier(this._repository)
      : super(const Tier1EditorState());

  final ProfileTier1Repository _repository;

  Future<bool> save(Location location) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.updateLocation(location);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('LocationEditorNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }

  void clearError() {
    if (state.error == null) return;
    state = state.copyWith(clearError: true);
  }
}

final locationEditorProvider =
    StateNotifierProvider<LocationEditorNotifier, Tier1EditorState>((ref) {
  final repo = ref.watch(profileTier1RepositoryProvider);
  return LocationEditorNotifier(repo);
});

// ---------------------------------------------------------------------------
// Languages notifier
// ---------------------------------------------------------------------------

class LanguagesEditorNotifier extends StateNotifier<Tier1EditorState> {
  LanguagesEditorNotifier(this._repository)
      : super(const Tier1EditorState());

  final ProfileTier1Repository _repository;

  Future<bool> save(
    List<String> professional,
    List<String> conversational,
  ) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.updateLanguages(professional, conversational);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('LanguagesEditorNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final languagesEditorProvider =
    StateNotifierProvider<LanguagesEditorNotifier, Tier1EditorState>((ref) {
  final repo = ref.watch(profileTier1RepositoryProvider);
  return LanguagesEditorNotifier(repo);
});

// ---------------------------------------------------------------------------
// Availability notifier
// ---------------------------------------------------------------------------

class AvailabilityEditorNotifier extends StateNotifier<Tier1EditorState> {
  AvailabilityEditorNotifier(this._repository)
      : super(const Tier1EditorState());

  final ProfileTier1Repository _repository;

  /// Patches one or both availability slots. At least one argument
  /// must be non-null — callers decide which slot they own based on
  /// the screen variant (direct profile vs referrer profile).
  Future<bool> save({
    AvailabilityStatus? direct,
    AvailabilityStatus? referrer,
  }) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.updateAvailability(direct: direct, referrer: referrer);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('AvailabilityEditorNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final availabilityEditorProvider =
    StateNotifierProvider<AvailabilityEditorNotifier, Tier1EditorState>((ref) {
  final repo = ref.watch(profileTier1RepositoryProvider);
  return AvailabilityEditorNotifier(repo);
});

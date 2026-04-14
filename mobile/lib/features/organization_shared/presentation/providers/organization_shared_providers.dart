import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/organization_shared_repository_impl.dart';
import '../../domain/entities/organization_shared_profile.dart';
import '../../domain/repositories/organization_shared_repository.dart';

// ---------------------------------------------------------------------------
// Dependency wiring
// ---------------------------------------------------------------------------

/// Swappable DI seam for the organization-shared feature. Tests
/// override this provider with an in-memory fake.
final organizationSharedRepositoryProvider =
    Provider<OrganizationSharedRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return OrganizationSharedRepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Read provider
// ---------------------------------------------------------------------------

/// Fetches the current operator's shared organization profile. Uses
/// `autoDispose` so every return to the edit screen re-reads the
/// latest state (e.g. after the photo upload flow wrote a new URL).
final organizationSharedProvider = FutureProvider.autoDispose<
    OrganizationSharedProfile>((ref) async {
  final repo = ref.watch(organizationSharedRepositoryProvider);
  return repo.getShared();
});

// ---------------------------------------------------------------------------
// Editor state
// ---------------------------------------------------------------------------

/// Immutable snapshot used by every shared-field editor notifier.
@immutable
class OrganizationSharedEditorState {
  const OrganizationSharedEditorState({
    this.isSaving = false,
    this.error,
  });

  final bool isSaving;
  final String? error;

  OrganizationSharedEditorState copyWith({
    bool? isSaving,
    String? error,
    bool clearError = false,
  }) {
    return OrganizationSharedEditorState(
      isSaving: isSaving ?? this.isSaving,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

// ---------------------------------------------------------------------------
// Location notifier
// ---------------------------------------------------------------------------

class SharedLocationNotifier
    extends StateNotifier<OrganizationSharedEditorState> {
  SharedLocationNotifier(this._repository)
      : super(const OrganizationSharedEditorState());

  final OrganizationSharedRepository _repository;

  Future<bool> save({
    required String city,
    required String countryCode,
    double? latitude,
    double? longitude,
    required List<String> workMode,
    int? travelRadiusKm,
  }) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.updateLocation(
        city: city,
        countryCode: countryCode,
        latitude: latitude,
        longitude: longitude,
        workMode: workMode,
        travelRadiusKm: travelRadiusKm,
      );
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('SharedLocationNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final sharedLocationEditorProvider = StateNotifierProvider<
    SharedLocationNotifier, OrganizationSharedEditorState>((ref) {
  final repo = ref.watch(organizationSharedRepositoryProvider);
  return SharedLocationNotifier(repo);
});

// ---------------------------------------------------------------------------
// Languages notifier
// ---------------------------------------------------------------------------

class SharedLanguagesNotifier
    extends StateNotifier<OrganizationSharedEditorState> {
  SharedLanguagesNotifier(this._repository)
      : super(const OrganizationSharedEditorState());

  final OrganizationSharedRepository _repository;

  Future<bool> save({
    required List<String> professional,
    required List<String> conversational,
  }) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.updateLanguages(
        professional: professional,
        conversational: conversational,
      );
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('SharedLanguagesNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final sharedLanguagesEditorProvider = StateNotifierProvider<
    SharedLanguagesNotifier, OrganizationSharedEditorState>((ref) {
  final repo = ref.watch(organizationSharedRepositoryProvider);
  return SharedLanguagesNotifier(repo);
});

// ---------------------------------------------------------------------------
// Photo notifier
// ---------------------------------------------------------------------------

class SharedPhotoNotifier
    extends StateNotifier<OrganizationSharedEditorState> {
  SharedPhotoNotifier(this._repository)
      : super(const OrganizationSharedEditorState());

  final OrganizationSharedRepository _repository;

  Future<bool> save(String photoUrl) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.updatePhoto(photoUrl);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('SharedPhotoNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final sharedPhotoEditorProvider =
    StateNotifierProvider<SharedPhotoNotifier, OrganizationSharedEditorState>(
        (ref) {
  final repo = ref.watch(organizationSharedRepositoryProvider);
  return SharedPhotoNotifier(repo);
});

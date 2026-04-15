import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/freelance_profile_repository_impl.dart';
import '../../domain/entities/freelance_pricing.dart';
import '../../domain/entities/freelance_profile.dart';
import '../../domain/repositories/freelance_profile_repository.dart';

// ---------------------------------------------------------------------------
// Dependency wiring
// ---------------------------------------------------------------------------

final freelanceProfileRepositoryProvider =
    Provider<FreelanceProfileRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return FreelanceProfileRepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Read providers
// ---------------------------------------------------------------------------

/// Fetches the authenticated operator's freelance profile. Uses
/// `autoDispose` so every return to the screen re-reads the latest
/// state after an edit.
final freelanceProfileProvider =
    FutureProvider.autoDispose<FreelanceProfile>((ref) async {
  final repo = ref.watch(freelanceProfileRepositoryProvider);
  return repo.getMy();
});

/// Parameterized read-only public profile provider. Used by the
/// `/freelancers/:id` route.
final freelancePublicProfileProvider = FutureProvider.autoDispose
    .family<FreelanceProfile, String>((ref, orgId) async {
  final repo = ref.watch(freelanceProfileRepositoryProvider);
  return repo.getPublic(orgId);
});

/// Current pricing row (or null). Kept as a separate provider so
/// the pricing section can refresh without re-reading the entire
/// profile payload.
final freelancePricingProvider =
    FutureProvider.autoDispose<FreelancePricing?>((ref) async {
  final repo = ref.watch(freelanceProfileRepositoryProvider);
  return repo.getPricing();
});

// ---------------------------------------------------------------------------
// Editor state
// ---------------------------------------------------------------------------

@immutable
class FreelanceEditorState {
  const FreelanceEditorState({this.isSaving = false, this.error});

  final bool isSaving;
  final String? error;

  FreelanceEditorState copyWith({
    bool? isSaving,
    String? error,
    bool clearError = false,
  }) {
    return FreelanceEditorState(
      isSaving: isSaving ?? this.isSaving,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

// ---------------------------------------------------------------------------
// Core (title/about/video) notifier
// ---------------------------------------------------------------------------

class FreelanceCoreEditorNotifier extends StateNotifier<FreelanceEditorState> {
  FreelanceCoreEditorNotifier(this._repo) : super(const FreelanceEditorState());

  final FreelanceProfileRepository _repo;

  Future<bool> save({
    required String title,
    required String about,
    required String videoUrl,
  }) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.updateCore(title: title, about: about, videoUrl: videoUrl);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('FreelanceCoreEditorNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final freelanceCoreEditorProvider = StateNotifierProvider<
    FreelanceCoreEditorNotifier, FreelanceEditorState>((ref) {
  return FreelanceCoreEditorNotifier(
    ref.watch(freelanceProfileRepositoryProvider),
  );
});

// ---------------------------------------------------------------------------
// Availability notifier
// ---------------------------------------------------------------------------

class FreelanceAvailabilityNotifier
    extends StateNotifier<FreelanceEditorState> {
  FreelanceAvailabilityNotifier(this._repo)
      : super(const FreelanceEditorState());

  final FreelanceProfileRepository _repo;

  Future<bool> save(String wireValue) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.updateAvailability(wireValue);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint(
        'FreelanceAvailabilityNotifier.save failed: $error\n$stackTrace',
      );
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final freelanceAvailabilityEditorProvider = StateNotifierProvider<
    FreelanceAvailabilityNotifier, FreelanceEditorState>((ref) {
  return FreelanceAvailabilityNotifier(
    ref.watch(freelanceProfileRepositoryProvider),
  );
});

// ---------------------------------------------------------------------------
// Pricing notifier
// ---------------------------------------------------------------------------

class FreelancePricingNotifier extends StateNotifier<FreelanceEditorState> {
  FreelancePricingNotifier(this._repo) : super(const FreelanceEditorState());

  final FreelanceProfileRepository _repo;

  Future<bool> upsert(FreelancePricing pricing) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.upsertPricing(pricing);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('FreelancePricingNotifier.upsert failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }

  Future<bool> delete() async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.deletePricing();
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('FreelancePricingNotifier.delete failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final freelancePricingEditorProvider = StateNotifierProvider<
    FreelancePricingNotifier, FreelanceEditorState>((ref) {
  return FreelancePricingNotifier(
    ref.watch(freelanceProfileRepositoryProvider),
  );
});

// ---------------------------------------------------------------------------
// Video notifier
// ---------------------------------------------------------------------------

class FreelanceVideoNotifier extends StateNotifier<FreelanceEditorState> {
  FreelanceVideoNotifier(this._repo) : super(const FreelanceEditorState());

  final FreelanceProfileRepository _repo;

  Future<bool> upload(File file) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.uploadVideo(file);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('FreelanceVideoNotifier.upload failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }

  Future<bool> remove() async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.deleteVideo();
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('FreelanceVideoNotifier.remove failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final freelanceVideoEditorProvider = StateNotifierProvider<
    FreelanceVideoNotifier, FreelanceEditorState>((ref) {
  return FreelanceVideoNotifier(
    ref.watch(freelanceProfileRepositoryProvider),
  );
});

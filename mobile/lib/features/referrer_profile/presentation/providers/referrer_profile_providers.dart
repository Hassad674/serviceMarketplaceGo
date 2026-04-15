import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/referrer_profile_repository_impl.dart';
import '../../domain/entities/referrer_pricing.dart';
import '../../domain/entities/referrer_profile.dart';
import '../../domain/repositories/referrer_profile_repository.dart';

// ---------------------------------------------------------------------------
// Dependency wiring
// ---------------------------------------------------------------------------

final referrerProfileRepositoryProvider =
    Provider<ReferrerProfileRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ReferrerProfileRepositoryImpl(api);
});

// ---------------------------------------------------------------------------
// Read providers
// ---------------------------------------------------------------------------

final referrerProfileProvider =
    FutureProvider.autoDispose<ReferrerProfile>((ref) async {
  final repo = ref.watch(referrerProfileRepositoryProvider);
  return repo.getMy();
});

final referrerPublicProfileProvider = FutureProvider.autoDispose
    .family<ReferrerProfile, String>((ref, orgId) async {
  final repo = ref.watch(referrerProfileRepositoryProvider);
  return repo.getPublic(orgId);
});

final referrerPricingProvider =
    FutureProvider.autoDispose<ReferrerPricing?>((ref) async {
  final repo = ref.watch(referrerProfileRepositoryProvider);
  return repo.getPricing();
});

// ---------------------------------------------------------------------------
// Editor state
// ---------------------------------------------------------------------------

@immutable
class ReferrerEditorState {
  const ReferrerEditorState({this.isSaving = false, this.error});

  final bool isSaving;
  final String? error;

  ReferrerEditorState copyWith({
    bool? isSaving,
    String? error,
    bool clearError = false,
  }) {
    return ReferrerEditorState(
      isSaving: isSaving ?? this.isSaving,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

// ---------------------------------------------------------------------------
// Core notifier
// ---------------------------------------------------------------------------

class ReferrerCoreEditorNotifier extends StateNotifier<ReferrerEditorState> {
  ReferrerCoreEditorNotifier(this._repo) : super(const ReferrerEditorState());

  final ReferrerProfileRepository _repo;

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
      debugPrint('ReferrerCoreEditorNotifier.save failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final referrerCoreEditorProvider = StateNotifierProvider<
    ReferrerCoreEditorNotifier, ReferrerEditorState>((ref) {
  return ReferrerCoreEditorNotifier(
    ref.watch(referrerProfileRepositoryProvider),
  );
});

// ---------------------------------------------------------------------------
// Availability notifier
// ---------------------------------------------------------------------------

class ReferrerAvailabilityNotifier
    extends StateNotifier<ReferrerEditorState> {
  ReferrerAvailabilityNotifier(this._repo)
      : super(const ReferrerEditorState());

  final ReferrerProfileRepository _repo;

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
        'ReferrerAvailabilityNotifier.save failed: $error\n$stackTrace',
      );
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final referrerAvailabilityEditorProvider = StateNotifierProvider<
    ReferrerAvailabilityNotifier, ReferrerEditorState>((ref) {
  return ReferrerAvailabilityNotifier(
    ref.watch(referrerProfileRepositoryProvider),
  );
});

// ---------------------------------------------------------------------------
// Pricing notifier
// ---------------------------------------------------------------------------

class ReferrerPricingNotifier extends StateNotifier<ReferrerEditorState> {
  ReferrerPricingNotifier(this._repo) : super(const ReferrerEditorState());

  final ReferrerProfileRepository _repo;

  Future<bool> upsert(ReferrerPricing pricing) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.upsertPricing(pricing);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('ReferrerPricingNotifier.upsert failed: $error\n$stackTrace');
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
      debugPrint('ReferrerPricingNotifier.delete failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final referrerPricingEditorProvider = StateNotifierProvider<
    ReferrerPricingNotifier, ReferrerEditorState>((ref) {
  return ReferrerPricingNotifier(
    ref.watch(referrerProfileRepositoryProvider),
  );
});

// ---------------------------------------------------------------------------
// Video notifier
// ---------------------------------------------------------------------------

class ReferrerVideoNotifier extends StateNotifier<ReferrerEditorState> {
  ReferrerVideoNotifier(this._repo) : super(const ReferrerEditorState());

  final ReferrerProfileRepository _repo;

  Future<bool> upload(File file) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repo.uploadVideo(file);
      if (!mounted) return true;
      state = state.copyWith(isSaving: false, clearError: true);
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('ReferrerVideoNotifier.upload failed: $error\n$stackTrace');
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
      debugPrint('ReferrerVideoNotifier.remove failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }
}

final referrerVideoEditorProvider = StateNotifierProvider<
    ReferrerVideoNotifier, ReferrerEditorState>((ref) {
  return ReferrerVideoNotifier(
    ref.watch(referrerProfileRepositoryProvider),
  );
});

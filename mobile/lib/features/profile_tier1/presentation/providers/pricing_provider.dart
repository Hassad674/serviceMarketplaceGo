import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../domain/entities/pricing.dart';
import '../../domain/entities/pricing_kind.dart';
import '../../domain/repositories/profile_tier1_repository.dart';
import 'profile_tier1_providers.dart';

/// Immutable snapshot of the pricing editor state.
///
/// Mirrors the [ProfileSkillsState] pattern (AsyncValue for the
/// loaded list, boolean for in-flight saves, sentinel error code
/// translated by the widget layer).
@immutable
class PricingState {
  const PricingState({
    required this.pricings,
    this.isSaving = false,
    this.error,
  });

  final AsyncValue<List<Pricing>> pricings;
  final bool isSaving;
  final String? error;

  PricingState copyWith({
    AsyncValue<List<Pricing>>? pricings,
    bool? isSaving,
    String? error,
    bool clearError = false,
  }) {
    return PricingState(
      pricings: pricings ?? this.pricings,
      isSaving: isSaving ?? this.isSaving,
      error: clearError ? null : (error ?? this.error),
    );
  }
}

/// StateNotifier that owns the caller's pricing rows.
///
/// - Loads the current list on construction.
/// - Provides `upsert(row)` and `remove(kind)` for the editor.
/// - Refetches after every successful mutation so the state is
///   always anchored to the server truth. Optimistic updates are
///   unnecessary here because the list is tiny (0..2 rows) and
///   the PUT returns the canonical row.
class PricingNotifier extends StateNotifier<PricingState> {
  PricingNotifier(this._repository)
      : super(const PricingState(pricings: AsyncValue.loading())) {
    _load();
  }

  final ProfileTier1Repository _repository;

  Future<void> _load() async {
    try {
      final list = await _repository.getPricing();
      if (!mounted) return;
      state = state.copyWith(
        pricings: AsyncValue.data(list),
        clearError: true,
      );
    } catch (error, stackTrace) {
      if (!mounted) return;
      debugPrint('PricingNotifier._load failed: $error');
      state = state.copyWith(pricings: AsyncValue.error(error, stackTrace));
    }
  }

  Future<void> refresh() => _load();

  /// Upsert a pricing row. Returns true on success.
  Future<bool> upsert(Pricing pricing) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.upsertPricing(pricing);
      final fresh = await _repository.getPricing();
      if (!mounted) return true;
      state = state.copyWith(
        pricings: AsyncValue.data(fresh),
        isSaving: false,
        clearError: true,
      );
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('PricingNotifier.upsert failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }

  /// Delete the row for the given kind. Returns true on success.
  Future<bool> remove(PricingKind kind) async {
    state = state.copyWith(isSaving: true, clearError: true);
    try {
      await _repository.deletePricing(kind);
      final fresh = await _repository.getPricing();
      if (!mounted) return true;
      state = state.copyWith(
        pricings: AsyncValue.data(fresh),
        isSaving: false,
        clearError: true,
      );
      return true;
    } catch (error, stackTrace) {
      if (!mounted) return false;
      debugPrint('PricingNotifier.remove failed: $error\n$stackTrace');
      state = state.copyWith(isSaving: false, error: 'generic');
      return false;
    }
  }

  void clearError() {
    if (state.error == null) return;
    state = state.copyWith(clearError: true);
  }
}

final pricingProvider =
    StateNotifierProvider<PricingNotifier, PricingState>((ref) {
  final repo = ref.watch(profileTier1RepositoryProvider);
  return PricingNotifier(repo);
});

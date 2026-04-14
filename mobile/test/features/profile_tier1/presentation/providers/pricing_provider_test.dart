import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/availability_status.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/location.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/pricing_kind.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/repositories/profile_tier1_repository.dart';
import 'package:marketplace_mobile/features/profile_tier1/presentation/providers/pricing_provider.dart';

/// In-memory fake repository — the provider tests need full control
/// over return values and error injection.
class _FakeRepository implements ProfileTier1Repository {
  List<Pricing> pricings = <Pricing>[];
  Exception? getError;
  Exception? writeError;
  int upsertCalls = 0;
  int deleteCalls = 0;

  @override
  Future<List<Pricing>> getPricing() async {
    if (getError != null) throw getError!;
    return pricings;
  }

  @override
  Future<Pricing> upsertPricing(Pricing pricing) async {
    upsertCalls += 1;
    if (writeError != null) throw writeError!;
    pricings = [
      ...pricings.where((p) => p.kind != pricing.kind),
      pricing,
    ];
    return pricing;
  }

  @override
  Future<void> deletePricing(PricingKind kind) async {
    deleteCalls += 1;
    if (writeError != null) throw writeError!;
    pricings = pricings.where((p) => p.kind != kind).toList();
  }

  @override
  Future<void> updateLocation(Location location) async {}

  @override
  Future<void> updateLanguages(
    List<String> professional,
    List<String> conversational,
  ) async {}

  @override
  Future<void> updateAvailability(
    AvailabilityStatus direct,
    AvailabilityStatus? referrer,
  ) async {}
}

void main() {
  group('PricingNotifier', () {
    test('transitions loading -> data on successful load', () async {
      final repo = _FakeRepository()
        ..pricings = [
          const Pricing(
            kind: PricingKind.direct,
            type: PricingType.daily,
            minAmount: 50000,
            maxAmount: null,
            currency: 'EUR',
            note: '',
            negotiable: false,
          ),
        ];
      final notifier = PricingNotifier(repo);
      expect(notifier.state.pricings.isLoading, isTrue);

      await Future<void>.delayed(Duration.zero);

      expect(notifier.state.pricings.hasValue, isTrue);
      expect(notifier.state.pricings.value!.length, 1);
      expect(notifier.state.isSaving, isFalse);
      notifier.dispose();
    });

    test('transitions to error state on failed load', () async {
      final repo = _FakeRepository()..getError = Exception('boom');
      final notifier = PricingNotifier(repo);
      await Future<void>.delayed(Duration.zero);

      expect(notifier.state.pricings, isA<AsyncError<List<Pricing>>>());
      notifier.dispose();
    });

    test('upsert returns true and refetches the list on success', () async {
      final repo = _FakeRepository();
      final notifier = PricingNotifier(repo);
      await Future<void>.delayed(Duration.zero);

      final ok = await notifier.upsert(
        const Pricing(
          kind: PricingKind.direct,
          type: PricingType.hourly,
          minAmount: 7500,
          maxAmount: null,
          currency: 'EUR',
          note: '',
          negotiable: false,
        ),
      );

      expect(ok, isTrue);
      expect(repo.upsertCalls, 1);
      expect(notifier.state.pricings.value!.length, 1);
      expect(notifier.state.isSaving, isFalse);
      expect(notifier.state.error, isNull);
      notifier.dispose();
    });

    test('upsert returns false on failure and sets error sentinel', () async {
      final repo = _FakeRepository()..writeError = Exception('bad');
      final notifier = PricingNotifier(repo);
      await Future<void>.delayed(Duration.zero);

      final ok = await notifier.upsert(
        const Pricing(
          kind: PricingKind.direct,
          type: PricingType.daily,
          minAmount: 50000,
          maxAmount: null,
          currency: 'EUR',
          note: '',
          negotiable: false,
        ),
      );

      expect(ok, isFalse);
      expect(notifier.state.error, 'generic');
      notifier.dispose();
    });

    test('remove invokes delete and refetches', () async {
      final repo = _FakeRepository()
        ..pricings = [
          const Pricing(
            kind: PricingKind.direct,
            type: PricingType.daily,
            minAmount: 50000,
            maxAmount: null,
            currency: 'EUR',
            note: '',
            negotiable: false,
          ),
          const Pricing(
            kind: PricingKind.referral,
            type: PricingType.commissionPct,
            minAmount: 500,
            maxAmount: null,
            currency: 'pct',
            note: '',
            negotiable: false,
          ),
        ];
      final notifier = PricingNotifier(repo);
      await Future<void>.delayed(Duration.zero);

      final ok = await notifier.remove(PricingKind.referral);

      expect(ok, isTrue);
      expect(repo.deleteCalls, 1);
      expect(notifier.state.pricings.value!.length, 1);
      expect(
        notifier.state.pricings.value!.single.kind,
        PricingKind.direct,
      );
      notifier.dispose();
    });

    test('clearError resets the error sentinel', () async {
      final repo = _FakeRepository()..writeError = Exception('bad');
      final notifier = PricingNotifier(repo);
      await Future<void>.delayed(Duration.zero);
      await notifier.upsert(
        const Pricing(
          kind: PricingKind.direct,
          type: PricingType.daily,
          minAmount: 1000,
          maxAmount: null,
          currency: 'EUR',
          note: '',
          negotiable: false,
        ),
      );
      expect(notifier.state.error, isNotNull);

      notifier.clearError();
      expect(notifier.state.error, isNull);
      notifier.dispose();
    });
  });
}

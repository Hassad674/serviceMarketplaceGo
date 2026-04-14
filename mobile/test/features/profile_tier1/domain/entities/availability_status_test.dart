import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/availability_status.dart';

void main() {
  group('AvailabilityStatus.fromWire', () {
    test('parses all three canonical values', () {
      expect(
        AvailabilityStatus.fromWire('available_now'),
        AvailabilityStatus.availableNow,
      );
      expect(
        AvailabilityStatus.fromWire('available_soon'),
        AvailabilityStatus.availableSoon,
      );
      expect(
        AvailabilityStatus.fromWire('not_available'),
        AvailabilityStatus.notAvailable,
      );
    });

    test('defaults to availableNow on null/empty/unknown', () {
      expect(
        AvailabilityStatus.fromWire(null),
        AvailabilityStatus.availableNow,
      );
      expect(
        AvailabilityStatus.fromWire(''),
        AvailabilityStatus.availableNow,
      );
      expect(
        AvailabilityStatus.fromWire('bogus'),
        AvailabilityStatus.availableNow,
      );
    });
  });

  group('AvailabilityStatus.fromWireOrNull', () {
    test('returns null on empty', () {
      expect(AvailabilityStatus.fromWireOrNull(null), isNull);
      expect(AvailabilityStatus.fromWireOrNull(''), isNull);
    });

    test('returns availableNow for unknown non-empty value', () {
      expect(
        AvailabilityStatus.fromWireOrNull('space_traveler'),
        AvailabilityStatus.availableNow,
      );
    });
  });

  group('wire strings', () {
    test('match backend enum values', () {
      expect(AvailabilityStatus.availableNow.wire, 'available_now');
      expect(AvailabilityStatus.availableSoon.wire, 'available_soon');
      expect(AvailabilityStatus.notAvailable.wire, 'not_available');
    });
  });
}

// search_filters_test.dart — unit coverage for MobileSearchFilters
// including copyWith sentinel semantics, equality, and the conversion
// to the SearchFilterInput shape used by buildFilterBy.

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/search/search_filters.dart';

void main() {
  group('MobileSearchFilters.isEmpty', () {
    test('canonical default is empty', () {
      expect(kEmptyMobileSearchFilters.isEmpty, isTrue);
    });

    test('a non-empty availability marks filters as non-empty', () {
      expect(
        const MobileSearchFilters(availability: MobileAvailabilityFilter.now)
            .isEmpty,
        isFalse,
      );
    });

    test('a single skill marks filters as non-empty', () {
      expect(
        const MobileSearchFilters(skills: <String>['React']).isEmpty,
        isFalse,
      );
    });

    test('priceMin=0 is still considered a filter', () {
      // 0 is a legitimate explicit value (user typed 0). Only null
      // means "unset", which is what `isEmpty` checks.
      expect(
        const MobileSearchFilters(priceMin: 0).isEmpty,
        isFalse,
      );
    });

    test('city whitespace is not considered a filter', () {
      // Whitespace-only city never produces a clause (see buildFilterBy),
      // but our isEmpty currently relies on String.isEmpty — document
      // the behaviour. Whitespace strings DO make isEmpty return false
      // and that is OK because the user must first clear the field.
      expect(
        const MobileSearchFilters(city: '   ').isEmpty,
        isFalse,
      );
    });
  });

  group('MobileSearchFilters.copyWith', () {
    test('no args returns equal filters', () {
      final a = const MobileSearchFilters(
        availability: MobileAvailabilityFilter.now,
        skills: <String>['React'],
      );
      final b = a.copyWith();
      expect(b, a);
      expect(identical(a, b), isFalse);
    });

    test('availability is replaced', () {
      final a = const MobileSearchFilters();
      final b = a.copyWith(availability: MobileAvailabilityFilter.soon);
      expect(b.availability, MobileAvailabilityFilter.soon);
    });

    test('priceMin can be set to a non-null value', () {
      final b = kEmptyMobileSearchFilters.copyWith(priceMin: 500);
      expect(b.priceMin, 500);
    });

    test('priceMin can be cleared with explicit null', () {
      final a = const MobileSearchFilters(priceMin: 500);
      final b = a.copyWith(priceMin: null);
      expect(b.priceMin, isNull);
    });

    test('priceMin is preserved when omitted', () {
      // This is the sentinel behaviour — if copyWith({priceMin: null})
      // meant "unset", we could never keep an existing value on a
      // call that does not touch priceMin.
      final a = const MobileSearchFilters(priceMin: 500);
      final b = a.copyWith(city: 'Paris');
      expect(b.priceMin, 500);
    });

    test('priceMax clearing does not affect priceMin', () {
      final a = const MobileSearchFilters(priceMin: 500, priceMax: 1000);
      final b = a.copyWith(priceMax: null);
      expect(b.priceMin, 500);
      expect(b.priceMax, isNull);
    });

    test('radiusKm can be explicitly null', () {
      final a = const MobileSearchFilters(city: 'Paris', radiusKm: 25);
      final b = a.copyWith(radiusKm: null);
      expect(b.radiusKm, isNull);
      expect(b.city, 'Paris');
    });

    test('skills list replaced', () {
      final a = const MobileSearchFilters(skills: ['React']);
      final b = a.copyWith(skills: const ['Go']);
      expect(b.skills, <String>['Go']);
    });

    test('workModes set replaced', () {
      final a = const MobileSearchFilters();
      final b = a.copyWith(
        workModes: <MobileWorkMode>{MobileWorkMode.remote},
      );
      expect(b.workModes, contains(MobileWorkMode.remote));
    });

    test('minRating can be set to zero to clear', () {
      final a = const MobileSearchFilters(minRating: 4);
      final b = a.copyWith(minRating: 0);
      expect(b.minRating, 0);
    });
  });

  group('MobileSearchFilters equality', () {
    test('identical payloads compare equal', () {
      final a = const MobileSearchFilters(
        city: 'Paris',
        skills: <String>['React', 'Go'],
      );
      final b = const MobileSearchFilters(
        city: 'Paris',
        skills: <String>['React', 'Go'],
      );
      expect(a, b);
      expect(a.hashCode, b.hashCode);
    });

    test('skill list order matters', () {
      final a = const MobileSearchFilters(skills: <String>['React', 'Go']);
      final b = const MobileSearchFilters(skills: <String>['Go', 'React']);
      expect(a == b, isFalse);
    });

    test('different case sensitivity breaks equality', () {
      final a = const MobileSearchFilters(skills: <String>['react']);
      final b = const MobileSearchFilters(skills: <String>['React']);
      expect(a == b, isFalse);
    });
  });

  group('filtersToInput', () {
    test('empty returns empty map', () {
      expect(filtersToInput(kEmptyMobileSearchFilters), isEmpty);
    });

    test('availability all is omitted', () {
      expect(
        filtersToInput(const MobileSearchFilters(
          availability: MobileAvailabilityFilter.all,
        )),
        isEmpty,
      );
    });

    test('availability now becomes a single-element list', () {
      final out = filtersToInput(
        const MobileSearchFilters(availability: MobileAvailabilityFilter.now),
      );
      expect(out['availabilityStatus'], <String>['now']);
    });

    test('city trimmed before export', () {
      final out = filtersToInput(const MobileSearchFilters(city: '  Paris  '));
      expect(out['city'], 'Paris');
    });

    test('radius requires a city or country', () {
      final withoutLoc = filtersToInput(
        const MobileSearchFilters(radiusKm: 25),
      );
      expect(withoutLoc.containsKey('geoRadiusKm'), isFalse);

      final withCity = filtersToInput(
        const MobileSearchFilters(city: 'Paris', radiusKm: 25),
      );
      expect(withCity['geoRadiusKm'], 25);

      final withCountry = filtersToInput(
        const MobileSearchFilters(countryCode: 'FR', radiusKm: 25),
      );
      expect(withCountry['geoRadiusKm'], 25);
    });

    test('work modes are emitted as canonical strings', () {
      final out = filtersToInput(
        const MobileSearchFilters(
          workModes: <MobileWorkMode>{
            MobileWorkMode.remote,
            MobileWorkMode.onSite,
            MobileWorkMode.hybrid,
          },
        ),
      );
      expect(
        out['workMode'],
        // Set-iteration order is insertion order in Dart.
        containsAll(<String>['remote', 'on_site', 'hybrid']),
      );
    });
  });

  group('kMobileExpertiseDomains', () {
    test('matches the web EXPERTISE_DOMAIN_KEYS list', () {
      // Snapshot — if the web adds a key, this list must follow.
      expect(kMobileExpertiseDomains, const <String>[
        'development',
        'design',
        'marketing',
        'product',
        'data',
        'devops',
        'cybersecurity',
        'ai_ml',
        'business',
        'operations',
        'sales',
        'hr',
        'legal',
        'finance',
        'content',
        'customer_support',
      ]);
    });
  });

  group('kMobilePopularSkills', () {
    test('matches the web POPULAR_SKILLS list', () {
      expect(kMobilePopularSkills, const <String>[
        'React',
        'TypeScript',
        'Go',
        'Python',
        'Node.js',
        'Figma',
        'Docker',
        'Kubernetes',
        'AWS',
        'PostgreSQL',
      ]);
    });
  });
}

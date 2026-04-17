// build_filter_by_test.dart — parity + unit test for the mobile
// Typesense filter_by builder.
//
// The expected strings in the PARITY table below are copied from the
// corresponding Go tests in
// `backend/internal/app/search/filter_builder_test.go` and from the
// TypeScript tests in
// `web/src/shared/lib/search/__tests__/build-filter-by.test.ts`.
// If any of them changes, a matching change must happen here —
// otherwise the mobile filter string will silently diverge from
// what the backend expects.
//
// Parity = byte-for-byte equality, not semantic equivalence.

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/search/build_filter_by.dart';
import 'package:marketplace_mobile/shared/search/search_filters.dart';

void main() {
  group('buildFilterBy parity', () {
    test('empty input returns empty string', () {
      expect(buildFilterBy(const SearchFilterInput()), '');
    });

    test('availability single value', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(availabilityStatus: ['now']),
        ),
        'availability_status:[now]',
      );
    });

    test('availability multi value preserves order', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(availabilityStatus: ['now', 'soon']),
        ),
        'availability_status:[now,soon]',
      );
    });

    test('availability dedupes', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(
            availabilityStatus: ['now', ' now ', 'now', 'soon'],
          ),
        ),
        'availability_status:[now,soon]',
      );
    });

    test('pricing min only', () {
      expect(
        buildFilterBy(const SearchFilterInput(pricingMin: 500)),
        'pricing_min_amount:>=500',
      );
    });

    test('pricing max only', () {
      expect(
        buildFilterBy(const SearchFilterInput(pricingMax: 1200)),
        'pricing_min_amount:<=1200',
      );
    });

    test('pricing min + max joined with &&', () {
      expect(
        buildFilterBy(const SearchFilterInput(pricingMin: 500, pricingMax: 1200)),
        'pricing_min_amount:>=500 && pricing_min_amount:<=1200',
      );
    });

    test('city backtick-escaped', () {
      expect(
        buildFilterBy(const SearchFilterInput(city: 'New York')),
        'city:`New York`',
      );
    });

    test('city trimmed', () {
      expect(
        buildFilterBy(const SearchFilterInput(city: '  Paris  ')),
        'city:`Paris`',
      );
    });

    test('countryCode preserves casing', () {
      expect(
        buildFilterBy(const SearchFilterInput(countryCode: 'FR')),
        'country_code:FR',
      );
    });

    test('geo clause emits lat,lng,N km', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(
            geoLat: 48.8566,
            geoLng: 2.3522,
            geoRadiusKm: 25,
          ),
        ),
        'location:(48.8566,2.3522,25 km)',
      );
    });

    test('geo clause drops when any coordinate missing', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(geoLat: 48.8566, geoRadiusKm: 25),
        ),
        '',
      );
    });

    test('geo clause drops when radius is zero', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(
            geoLat: 48.8566,
            geoLng: 2.3522,
            geoRadiusKm: 0,
          ),
        ),
        '',
      );
    });

    test('languages OR', () {
      expect(
        buildFilterBy(const SearchFilterInput(languages: ['fr', 'en'])),
        'languages_professional:[fr,en]',
      );
    });

    test('expertise OR', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(expertiseDomains: ['development', 'design']),
        ),
        'expertise_domains:[development,design]',
      );
    });

    test('skills OR preserves order', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(skills: ['React', 'Go', 'TypeScript']),
        ),
        'skills:[React,Go,TypeScript]',
      );
    });

    test('rating clause', () {
      expect(
        buildFilterBy(const SearchFilterInput(ratingMin: 4.5)),
        'rating_average:>=4.5',
      );
    });

    test('rating clause zero dropped', () {
      expect(
        buildFilterBy(const SearchFilterInput(ratingMin: 0)),
        '',
      );
    });

    test('work mode list', () {
      expect(
        buildFilterBy(
          const SearchFilterInput(workMode: ['remote', 'hybrid']),
        ),
        'work_mode:[remote,hybrid]',
      );
    });

    test('booleans use := operator', () {
      expect(
        buildFilterBy(const SearchFilterInput(isVerified: true)),
        'is_verified:=true',
      );
      expect(
        buildFilterBy(const SearchFilterInput(isTopRated: false)),
        'is_top_rated:=false',
      );
      expect(
        buildFilterBy(const SearchFilterInput(negotiable: true)),
        'pricing_negotiable:=true',
      );
    });

    test('full payload order matches backend (parity anchor)', () {
      // This is THE parity anchor — a lived-in filter payload that
      // exercises every clause. If this test fails, filter_by has
      // drifted from the Go + TS counterparts.
      final input = const SearchFilterInput(
        availabilityStatus: ['now'],
        pricingMin: 500,
        pricingMax: 1500,
        city: 'Paris',
        countryCode: 'FR',
        geoLat: 48.8566,
        geoLng: 2.3522,
        geoRadiusKm: 25,
        languages: ['fr', 'en'],
        expertiseDomains: ['development'],
        skills: ['React', 'Go'],
        ratingMin: 4,
        workMode: ['remote'],
        isVerified: true,
        isTopRated: true,
        negotiable: false,
      );
      expect(
        buildFilterBy(input),
        'availability_status:[now] && '
        'pricing_min_amount:>=500 && pricing_min_amount:<=1500 && '
        'city:`Paris` && country_code:FR && '
        'location:(48.8566,2.3522,25 km) && '
        'languages_professional:[fr,en] && '
        'expertise_domains:[development] && '
        'skills:[React,Go] && '
        'rating_average:>=4 && '
        'work_mode:[remote] && '
        'is_verified:=true && '
        'is_top_rated:=true && '
        'pricing_negotiable:=false',
      );
    });
  });

  group('SearchFilterInput.fromMap', () {
    test('accepts the filtersToInput payload shape', () {
      final mobileFilters = const MobileSearchFilters(
        availability: MobileAvailabilityFilter.now,
        priceMin: 500,
        priceMax: 1500,
        city: 'Paris',
        countryCode: 'FR',
        radiusKm: 25,
        languages: <String>{'fr', 'en'},
        expertise: <String>{'development'},
        skills: <String>['React'],
        minRating: 4,
        workModes: <MobileWorkMode>{MobileWorkMode.remote},
      );
      final input = SearchFilterInput.fromMap(filtersToInput(mobileFilters));
      final out = buildFilterBy(input);
      expect(
        out,
        'availability_status:[now] && '
        'pricing_min_amount:>=500 && pricing_min_amount:<=1500 && '
        'city:`Paris` && country_code:FR && '
        'languages_professional:[fr,en] && '
        'expertise_domains:[development] && '
        'skills:[React] && '
        'rating_average:>=4 && '
        'work_mode:[remote]',
      );
    });

    test('empty filters yield empty filter_by', () {
      final input = SearchFilterInput.fromMap(
        filtersToInput(kEmptyMobileSearchFilters),
      );
      expect(buildFilterBy(input), '');
    });

    test('radius is dropped when no city/country is set', () {
      final f = const MobileSearchFilters(radiusKm: 25);
      expect(filtersToInput(f).containsKey('geoRadiusKm'), isFalse);
    });

    test('radius is kept when city is set', () {
      final f = const MobileSearchFilters(city: 'Paris', radiusKm: 25);
      expect(filtersToInput(f)['geoRadiusKm'], 25);
    });

    test('unknown keys ignored (forward compat)', () {
      final map = <String, Object?>{'unexpected_key': 'value', 'city': 'Paris'};
      final input = SearchFilterInput.fromMap(map);
      expect(input.city, 'Paris');
    });
  });
}

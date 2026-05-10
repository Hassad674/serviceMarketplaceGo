import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/search/profile_route_builder.dart';

void main() {
  group('buildProfileRouteFromSearch', () {
    test('routes freelance persona to /freelancers/<id>', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc-123',
        persona: 'freelance',
      );
      expect(url, '/freelancers/abc-123');
    });

    test('routes referrer persona to /referrers/<id>', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'org-1',
        persona: 'referrer',
      );
      expect(url, '/referrers/org-1');
    });

    test('routes client persona to /clients/<id>', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'c1',
        persona: 'client',
      );
      expect(url, '/clients/c1');
    });

    test('routes agency persona to legacy /profiles/<id>', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'agc',
        persona: 'agency',
      );
      expect(url, '/profiles/agc');
    });

    test('unknown persona falls back to /profiles/<id>', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'mystery',
        persona: 'wat',
      );
      expect(url, '/profiles/mystery');
    });

    test('appends q + pos when both are provided', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: 'designer',
        position: 3,
      );
      expect(url, '/freelancers/abc?q=designer&pos=3');
    });

    test('lowercases + trims the query', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: '  Senior Designer  ',
        position: 1,
      );
      expect(url, '/freelancers/abc?q=senior+designer&pos=1');
    });

    test('omits empty query', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: '   ',
        position: 2,
      );
      expect(url, '/freelancers/abc?pos=2');
    });

    test('omits position when null', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: 'go',
      );
      expect(url, '/freelancers/abc?q=go');
    });

    test('omits position when below 1', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: 'go',
        position: 0,
      );
      expect(url, '/freelancers/abc?q=go');
    });

    test('encodes special characters in orgId', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'a/b c',
        persona: 'freelance',
      );
      expect(url, '/freelancers/a%2Fb%20c');
    });

    test('percent-encodes query special chars (+ for space)', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: 'c++ guru',
        position: 5,
      );
      // Uri.encodeQueryComponent uses `+` for space and percent-encodes
      // `+` itself.
      expect(url, '/freelancers/abc?q=c%2B%2B+guru&pos=5');
    });

    test('returns clean path when neither query nor position supplied',
        () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: '',
        position: null,
      );
      expect(url, '/freelancers/abc');
    });
  });
}

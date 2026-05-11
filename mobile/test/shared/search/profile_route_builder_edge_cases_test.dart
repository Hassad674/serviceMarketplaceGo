// Extra edge-case coverage for [buildProfileRouteFromSearch] sitting
// alongside the main `profile_route_builder_test.dart`. The original
// file covers the happy paths + 13 nominal cases; this companion adds
// the gnarly inputs surfaced during testing-coverage review:
//
//   * very long query strings (≥ 256 chars)
//   * Unicode + non-ASCII (accented chars, emoji)
//   * negative position (must be omitted)
//   * persona is case-sensitive (uppercase variant falls back)
//   * trailing/leading whitespace + tabs/newlines collapse
//   * empty orgId edge (degenerate but must not crash)

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/shared/search/profile_route_builder.dart';

void main() {
  group('buildProfileRouteFromSearch — edge cases', () {
    test('handles a 512-char query without overflowing or crashing', () {
      final longQuery = List.filled(512, 'a').join();
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: longQuery,
        position: 1,
      );
      expect(url.startsWith('/freelancers/abc?q='), isTrue);
      // Position still suffixed despite the long query.
      expect(url.endsWith('&pos=1'), isTrue);
    });

    test('handles accented Unicode in the query', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: 'Développeur Sénior',
      );
      // Uri.encodeQueryComponent percent-encodes the accents — assert
      // the path stays well-formed and lowercased before encoding.
      expect(url, contains('/freelancers/abc?q='));
      // Lowercased: `développeur sénior` then percent-encoded.
      expect(url.toLowerCase(), contains('d%c3%a9veloppeur'));
    });

    test('handles emoji in the query', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: 'rocket 🚀 dev',
      );
      expect(url, startsWith('/freelancers/abc?q='));
      // Emoji is multi-byte → percent-encoded → length > base path.
      expect(url.length, greaterThan('/freelancers/abc?q='.length + 10));
    });

    test('negative position is omitted (treated like null)', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: 'go',
        position: -5,
      );
      expect(url, '/freelancers/abc?q=go');
    });

    test('persona is case-sensitive — uppercase falls back to /profiles', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'xyz',
        persona: 'Freelance',
      );
      expect(url, '/profiles/xyz');
    });

    test('empty persona falls back to /profiles', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'xyz',
        persona: '',
      );
      expect(url, '/profiles/xyz');
    });

    test('trims tabs and newlines from the query', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: '\t\n  Go  \n',
        position: 4,
      );
      // The implementation `.trim()`s before lowercasing; tabs / newlines
      // are stripped, the body is "Go" → "go".
      expect(url, '/freelancers/abc?q=go&pos=4');
    });

    test('mixed-case persona variants ("FREELANCE") fall back', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'FREELANCE',
      );
      expect(url, '/profiles/abc');
    });

    test('position 1 (first result) is preserved (boundary)', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        position: 1,
      );
      expect(url, '/freelancers/abc?pos=1');
    });

    test('position immediately above 1 is preserved', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        position: 2,
      );
      expect(url, '/freelancers/abc?pos=2');
    });

    test('handles empty orgId gracefully (degenerate but not a crash)', () {
      final url = buildProfileRouteFromSearch(
        orgId: '',
        persona: 'freelance',
      );
      expect(url, '/freelancers/');
    });

    test('query with only special chars survives encoding', () {
      final url = buildProfileRouteFromSearch(
        orgId: 'abc',
        persona: 'freelance',
        query: '?&=#',
      );
      expect(url, startsWith('/freelancers/abc?q='));
      // None of the reserved chars leak into the URL un-encoded.
      expect(url.contains('?&=#'), isFalse);
    });
  });
}

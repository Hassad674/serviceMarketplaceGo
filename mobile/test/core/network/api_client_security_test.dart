// SEC-08 — Unit tests for the API URL safety guard.
//
// We can't assert on the real `ApiClient` constructor without standing
// up FlutterSecureStorage; the rule has been extracted into a pure
// top-level function `isApiUrlSafeForReleaseBuild` precisely so it can
// be exercised with cheap, table-driven tests like the ones below.

import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';

void main() {
  group('isApiUrlSafeForReleaseBuild', () {
    test('debug build accepts any URL (HTTP, HTTPS, malformed)', () {
      const cases = [
        'http://10.0.2.2:8083',
        'http://localhost:8083',
        'http://192.168.1.156:8083',
        'https://api.marketplace.example.com',
        // Even completely malformed URLs pass in debug — we only care
        // about the production constraint.
        'not-a-url',
        '',
      ];
      for (final url in cases) {
        expect(
          isApiUrlSafeForReleaseBuild(url, isDebug: true),
          isTrue,
          reason: 'debug mode must accept "$url"',
        );
      }
    });

    test('release build rejects http:// URLs', () {
      const cases = [
        'http://10.0.2.2:8083',
        'http://localhost:8083',
        'http://192.168.1.156:8083',
        'http://api.marketplace.example.com',
      ];
      for (final url in cases) {
        expect(
          isApiUrlSafeForReleaseBuild(url, isDebug: false),
          isFalse,
          reason: 'release mode must REJECT "$url"',
        );
      }
    });

    test('release build accepts https:// URLs', () {
      const cases = [
        'https://api.marketplace.example.com',
        'https://staging.marketplace.example.com',
        'https://localhost:8083', // unlikely but still safe
      ];
      for (final url in cases) {
        expect(
          isApiUrlSafeForReleaseBuild(url, isDebug: false),
          isTrue,
          reason: 'release mode must accept "$url"',
        );
      }
    });

    test('release build rejects empty / malformed URLs', () {
      const cases = [
        '',
        'no-scheme.example.com',
        'ftp://example.com',
        'ws://example.com',
        ' https://example.com', // leading whitespace
      ];
      for (final url in cases) {
        expect(
          isApiUrlSafeForReleaseBuild(url, isDebug: false),
          isFalse,
          reason: 'release mode must reject "$url"',
        );
      }
    });

    test('https:// prefix check is case-sensitive (per RFC 3986 §3.1)', () {
      // RFC 3986 says the scheme should be normalized to lowercase by
      // the parser, but the value we check is what the developer wrote
      // at build time. A typo like "Https://" would only happen if the
      // developer typed it themselves — we reject it so the developer
      // gets a loud failure.
      expect(
        isApiUrlSafeForReleaseBuild('Https://example.com', isDebug: false),
        isFalse,
      );
      expect(
        isApiUrlSafeForReleaseBuild('HTTPS://example.com', isDebug: false),
        isFalse,
      );
    });
  });
}

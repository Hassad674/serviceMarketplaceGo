// H2/M3 — Unit tests for cert pinning logic.
//
// We can't drive a real TLS handshake from a unit test, so the design
// of cert_pinning_interceptor.dart deliberately splits the policy
// (pure functions) from the I/O wiring (`installPinningOnAdapter`,
// which is exercised in integration). Tests below cover:
//
//   1. `shouldEnforceCertPinning` activation matrix (build mode +
//      URL + pin presence).
//   2. `isLanOrLoopbackHost` LAN detection edge cases.
//   3. `normalizePins` accepts colon-separated and bare-hex inputs.
//   4. `sha256HexOfBytes` matches FIPS 180-4 reference vectors.
//   5. `verifyCertAgainstPins` returns the right [PinningDecision]
//      for match/mismatch/empty cases (using a synthetic cert).
//   6. [CertPinningInterceptor] forwards responses and exposes its
//      configured pin set.
//   7. [CertificatePinningException.toString] does NOT leak the
//      cert hash (security: hash must not appear in crash reports).

import 'dart:io';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/cert_pinning_interceptor.dart';

void main() {
  group('shouldEnforceCertPinning', () {
    test('debug build is always bypassed', () {
      // Even with both pins set and a prod URL, debug builds skip
      // pinning so devs are never bricked by a typoed pin.
      expect(
        shouldEnforceCertPinning(
          'https://api.example.com',
          isRelease: false,
          primaryPin: 'aaaa',
          backupPin: 'bbbb',
        ),
        isFalse,
      );
    });

    test('release without any pins is bypassed (graceful degradation)', () {
      // Empty pins → degrade to OS trust. This is intentional: a
      // missed CI step should not brick the app.
      expect(
        shouldEnforceCertPinning(
          'https://api.example.com',
          isRelease: true,
          primaryPin: '',
          backupPin: '',
        ),
        isFalse,
      );
    });

    test('release + non-https URL is bypassed', () {
      // SEC-08 already prevents shipping a release that talks HTTP,
      // but pinning must not panic if it gets one anyway.
      expect(
        shouldEnforceCertPinning(
          'http://api.example.com',
          isRelease: true,
          primaryPin: 'aaaa',
          backupPin: 'bbbb',
        ),
        isFalse,
      );
    });

    test('release + LAN URL is bypassed', () {
      const lanUrls = [
        'https://192.168.1.5',
        'https://10.0.0.1',
        'https://172.16.5.4',
        'https://localhost:8083',
        'https://127.0.0.1',
        'https://10.0.2.2',
      ];
      for (final url in lanUrls) {
        expect(
          shouldEnforceCertPinning(
            url,
            isRelease: true,
            primaryPin: 'aaaa',
            backupPin: 'bbbb',
          ),
          isFalse,
          reason: 'release + LAN ($url) must be bypassed',
        );
      }
    });

    test('release + prod https + at least one pin → enforced', () {
      // Primary only.
      expect(
        shouldEnforceCertPinning(
          'https://api.atelier.example.com',
          isRelease: true,
          primaryPin: 'aaaa',
          backupPin: '',
        ),
        isTrue,
      );
      // Backup only — also enforced.
      expect(
        shouldEnforceCertPinning(
          'https://api.atelier.example.com',
          isRelease: true,
          primaryPin: '',
          backupPin: 'bbbb',
        ),
        isTrue,
      );
      // Both pins.
      expect(
        shouldEnforceCertPinning(
          'https://api.atelier.example.com',
          isRelease: true,
          primaryPin: 'aaaa',
          backupPin: 'bbbb',
        ),
        isTrue,
      );
    });

    test('release + malformed URL is bypassed (parse fails)', () {
      // Uri.tryParse returns non-null for many malformed strings,
      // but the scheme check still rejects them. Either way we must
      // NOT enable pinning against a URL we cannot parse.
      expect(
        shouldEnforceCertPinning(
          'not://a real ::: url',
          isRelease: true,
          primaryPin: 'aaaa',
          backupPin: 'bbbb',
        ),
        isFalse,
      );
    });
  });

  group('isLanOrLoopbackHost', () {
    test('classic loopback hosts', () {
      expect(isLanOrLoopbackHost('localhost'), isTrue);
      expect(isLanOrLoopbackHost('127.0.0.1'), isTrue);
      expect(isLanOrLoopbackHost('::1'), isTrue);
      expect(isLanOrLoopbackHost('10.0.2.2'), isTrue);
    });

    test('RFC 1918 ranges', () {
      expect(isLanOrLoopbackHost('10.0.0.5'), isTrue);
      expect(isLanOrLoopbackHost('192.168.1.1'), isTrue);
      expect(isLanOrLoopbackHost('172.16.0.1'), isTrue);
      expect(isLanOrLoopbackHost('172.31.255.255'), isTrue);
    });

    test('172.x.x.x outside 16..31 is NOT private', () {
      // 172.15.x.x and 172.32.x.x are public.
      expect(isLanOrLoopbackHost('172.15.0.1'), isFalse);
      expect(isLanOrLoopbackHost('172.32.0.1'), isFalse);
    });

    test('public IPs and DNS names are NOT LAN', () {
      expect(isLanOrLoopbackHost('api.atelier.example.com'), isFalse);
      expect(isLanOrLoopbackHost('1.1.1.1'), isFalse);
      expect(isLanOrLoopbackHost('8.8.8.8'), isFalse);
    });

    test('empty host is treated as LAN (defensive)', () {
      // Cannot pin against an empty host; safer to bypass than to
      // accept arbitrary certs.
      expect(isLanOrLoopbackHost(''), isTrue);
    });
  });

  group('normalizePins', () {
    test('strips colons, whitespace, lowercases', () {
      final out = normalizePins([
        'AA:BB:CC:DD',
        'aabbccdd',
        ' AA BB CC DD ',
        '',
      ]);
      // All three first inputs collapse to the same 8-char hex; the
      // empty entry is dropped.
      expect(out, equals({'aabbccdd'}));
    });

    test('keeps distinct fingerprints distinct', () {
      final out = normalizePins(['AA:BB', 'CC:DD']);
      expect(out, equals({'aabb', 'ccdd'}));
    });

    test('empty input → empty set', () {
      expect(normalizePins([]), isEmpty);
      expect(normalizePins(['', '', '']), isEmpty);
    });
  });

  group('sha256HexOfBytes', () {
    test('matches FIPS 180-4 vector for empty input', () {
      // SHA-256 of "" = e3b0c44298fc1c149afbf4c8996fb924…
      expect(
        sha256HexOfBytes(<int>[]),
        equals(
            'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855'),
      );
    });

    test('matches FIPS 180-4 vector for "abc"', () {
      // SHA-256 of "abc" = ba7816bf8f01cfea414140de5dae2223…
      expect(
        sha256HexOfBytes('abc'.codeUnits),
        equals(
            'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad'),
      );
    });

    test('matches FIPS 180-4 vector for the 56-byte boundary string', () {
      // 'abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq' (56 bytes)
      // produces 248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167…
      const input =
          'abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq';
      expect(
        sha256HexOfBytes(input.codeUnits),
        equals(
            '248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1'),
      );
    });
  });

  group('verifyCertAgainstPins', () {
    test('empty pin set → notConfigured', () {
      final cert = _fakeCert(<int>[1, 2, 3]);
      expect(
        verifyCertAgainstPins(cert, <String>{}),
        equals(PinningDecision.notConfigured),
      );
    });

    test('matching pin → match', () {
      final bytes = <int>[1, 2, 3, 4, 5];
      final cert = _fakeCert(bytes);
      final pin = sha256HexOfBytes(bytes);
      expect(
        verifyCertAgainstPins(cert, <String>{pin}),
        equals(PinningDecision.match),
      );
    });

    test('non-matching pin → mismatch', () {
      final cert = _fakeCert(<int>[1, 2, 3]);
      expect(
        verifyCertAgainstPins(
          cert,
          <String>{'deadbeefdeadbeefdeadbeefdeadbeef' * 2},
        ),
        equals(PinningDecision.mismatch),
      );
    });

    test('pin set with multiple entries — match on either', () {
      final bytes = <int>[10, 20, 30];
      final cert = _fakeCert(bytes);
      final pin = sha256HexOfBytes(bytes);
      expect(
        verifyCertAgainstPins(
          cert,
          <String>{
            'deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
            pin,
          },
        ),
        equals(PinningDecision.match),
      );
    });
  });

  group('CertPinningInterceptor', () {
    test('exposes the normalised pin set', () {
      final interceptor = CertPinningInterceptor(
        acceptedPins: {'AA:BB:CC', 'ddee', ''},
      );
      expect(interceptor.acceptedPins, equals({'aabbcc', 'ddee'}));
    });

    test('onResponse forwards the response (belt-and-braces guard only)',
        () async {
      final interceptor = CertPinningInterceptor(
        acceptedPins: {'aabbcc'},
      );
      final response = Response<dynamic>(
        requestOptions: RequestOptions(path: '/api/v1/health'),
        statusCode: 200,
      );
      final handler = _CapturingResponseHandler();
      interceptor.onResponse(response, handler);
      expect(handler.captured, same(response));
    });
  });

  group('CertificatePinningException', () {
    test('toString does NOT leak the live cert hash', () {
      // Security: no live fingerprint must appear in any error
      // surface. The exception only carries the host + a generic
      // message — both safe to log.
      const ex = CertificatePinningException(
        'server cert fingerprint did not match any configured pin',
        'api.example.com',
      );
      final s = ex.toString();
      expect(s, contains('api.example.com'));
      expect(s, contains('did not match'));
      // Sanity: no hex blob looking like a fingerprint should sneak
      // in. We check there's no 64-char lowercase-hex run.
      expect(
        RegExp(r'[0-9a-f]{32,}').hasMatch(s),
        isFalse,
        reason: 'exception.toString must not embed a fingerprint',
      );
    });
  });

  group('defaultProductionPins', () {
    test('returns empty set when neither dart-define is set', () {
      // The unit-test process is launched without
      // BACKEND_CERT_SHA256_PRIMARY/BACKUP, so both compile-time
      // constants resolve to ''. The function must drop them.
      expect(defaultProductionPins(), isEmpty);
    });
  });
}

/// Builds a minimal fake [X509Certificate] backed by the given DER
/// bytes. Used by `verifyCertAgainstPins` tests that need a
/// fingerprint computed from a known input without standing up a
/// real TLS server.
X509Certificate _fakeCert(List<int> der) =>
    _FakeX509Certificate(Uint8List.fromList(der));

class _FakeX509Certificate implements X509Certificate {
  _FakeX509Certificate(this._der);

  final Uint8List _der;

  @override
  Uint8List get der => _der;

  @override
  Uint8List get sha1 => throw UnimplementedError('not used by tests');

  @override
  String get pem => throw UnimplementedError('not used by tests');

  @override
  String get subject => 'CN=fake';

  @override
  String get issuer => 'CN=fake';

  @override
  DateTime get startValidity => DateTime(2026);

  @override
  DateTime get endValidity => DateTime(2027);
}

/// Captures the response handed to handler.next so tests can assert
/// against it without instantiating a real Dio.
class _CapturingResponseHandler extends ResponseInterceptorHandler {
  Response<dynamic>? captured;

  @override
  void next(Response<dynamic> response) {
    captured = response;
    super.next(response);
  }
}

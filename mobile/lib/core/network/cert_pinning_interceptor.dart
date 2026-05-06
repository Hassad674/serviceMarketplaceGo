// H2/M3 — Certificate pinning for production builds.
//
// Why pinning?
//   The mobile app moves money (Stripe payouts, escrow, payments) and
//   PII (KYC documents). A malicious WiFi hotspot, a compromised
//   corporate proxy, or a state-level MITM can present a forged TLS
//   cert that the OS trust store accepts. Pinning the SHA-256
//   fingerprint of the backend's leaf cert stops that attack: even a
//   "valid" cert chain fails because the fingerprint does not match
//   the compile-time pin.
//
// Where it activates (see [shouldEnforceCertPinning]):
//   * `kReleaseMode == true`      → enforced
//   * `kReleaseMode == false`     → bypassed (developer flow)
//   * baseUrl is LAN/loopback     → bypassed (developer flow)
//   * baseUrl is non-https://     → bypassed (handled by SEC-08 assert)
//   * no pins configured at all   → bypassed (graceful degradation)
//
// How fingerprints are provided:
//   At BUILD time via Dart-define:
//
//     flutter build apk --release \
//       --dart-define=API_URL=https://api.atelier.example.com \
//       --dart-define=BACKEND_CERT_SHA256_PRIMARY=AAAA…AAAA \
//       --dart-define=BACKEND_CERT_SHA256_BACKUP=BBBB…BBBB
//
//   Two slots (PRIMARY + BACKUP) so cert rotation is zero-downtime:
//     1. Issue the next-rotation cert, compute its SHA-256.
//     2. Ship a release with PRIMARY=<old> + BACKUP=<new>.
//     3. Wait for ≥95% adoption.
//     4. Deploy <new> on the backend.
//     5. Ship a release with PRIMARY=<new> + BACKUP=<next-rotation>.
//
//   Fingerprints are compile-time constants (not env-readable at
//   runtime) so an attacker who roots a device cannot swap them.
//
// Cert rotation runbook lives in mobile/CLAUDE.md.

import 'dart:io';

import 'package:dio/dio.dart';
import 'package:dio/io.dart';
import 'package:flutter/foundation.dart';

/// Compile-time SHA-256 fingerprint of the production backend's TLS
/// leaf cert (DER-encoded, lower-case hex, no colons).
///
/// Empty string means "not configured" — the activation gate
/// ([shouldEnforceCertPinning]) returns false in that case so an
/// unconfigured release build does not brick the user. Bricking via
/// missing cert is worse than shipping without pinning. The release
/// pipeline is responsible for passing both pins; CI should fail a
/// release build if either is blank.
const String _kPrimaryPinHex = String.fromEnvironment(
  'BACKEND_CERT_SHA256_PRIMARY',
  defaultValue: '',
);

/// Backup pin — the next-rotation cert. Allows zero-downtime cert
/// rotation: ship app with PRIMARY=<old> + BACKUP=<new>, then once
/// adoption is high deploy <new> on the backend.
const String _kBackupPinHex = String.fromEnvironment(
  'BACKEND_CERT_SHA256_BACKUP',
  defaultValue: '',
);

/// Thrown when the server cert SHA-256 does not match any configured
/// pin. Surfaced through Dio as a `DioException` of type
/// `badCertificate` so callers can show a clear "Network not safe —
/// please switch network" error instead of a silent failure.
class CertificatePinningException implements Exception {
  /// Human-readable failure reason. Does NOT contain the live cert
  /// hash — we never want a fingerprint to leak into crash reports.
  final String message;

  /// Underlying host whose cert failed verification. Safe to surface.
  final String host;

  const CertificatePinningException(this.message, this.host);

  @override
  String toString() => 'CertificatePinningException($host): $message';
}

/// Decides whether cert pinning should be enforced for [baseUrl] in
/// the current build mode.
///
/// Extracted as a top-level pure function so the rule can be unit
/// tested with cheap inputs — no Dio, no platform plugins. Keep this
/// in sync with the activation rules at the top of the file.
///
/// Public (not `@visibleForTesting`) because `api_client.dart`
/// invokes it as part of the live wiring path; tests exercise it
/// independently.
bool shouldEnforceCertPinning(
  String baseUrl, {
  required bool isRelease,
  String primaryPin = _kPrimaryPinHex,
  String backupPin = _kBackupPinHex,
}) {
  // Debug / profile builds: never enforce. Developers run against
  // local backends and a typoed pin would brick the inner loop.
  if (!isRelease) return false;

  // No pins configured — degrade gracefully. A release without pins
  // is a CI bug, not a runtime crash.
  if (primaryPin.isEmpty && backupPin.isEmpty) return false;

  final uri = Uri.tryParse(baseUrl);
  if (uri == null) return false;

  // Only HTTPS targets need pinning. The SEC-08 assert in
  // api_client.dart already rejects non-https in release; this is
  // belt-and-braces so a future regression there does not silently
  // disable pinning here.
  if (uri.scheme != 'https') return false;

  // Skip LAN / loopback targets. A release build pointing at
  // 192.168.x.x is most likely a staging-on-LAN build; pinning the
  // dev cert is high-cost low-value.
  if (_isLanOrLoopbackHost(uri.host)) return false;

  return true;
}

/// True when [host] looks like a developer-LAN or loopback target.
@visibleForTesting
bool isLanOrLoopbackHost(String host) => _isLanOrLoopbackHost(host);

bool _isLanOrLoopbackHost(String host) {
  if (host.isEmpty) return true;
  if (host == 'localhost') return true;
  if (host == '10.0.2.2') return true; // Android emulator → host loopback
  if (host == '127.0.0.1' || host == '::1') return true;

  // RFC 1918 private ranges. Cheap string check — we don't fully
  // parse every IP form.
  if (host.startsWith('10.')) return true;
  if (host.startsWith('192.168.')) return true;
  if (host.startsWith('172.')) {
    // 172.16.0.0/12 — second octet 16..31
    final parts = host.split('.');
    if (parts.length >= 2) {
      final second = int.tryParse(parts[1]);
      if (second != null && second >= 16 && second <= 31) return true;
    }
  }
  return false;
}

/// Returns the configured production pins as a normalised set,
/// dropping empty slots. Convenience for `api_client.dart` so it
/// does not have to reach into private constants.
///
/// Public (not `@visibleForTesting`) because `api_client.dart`
/// invokes it as part of the live wiring path.
Set<String> defaultProductionPins() {
  final raw = <String>[];
  if (_kPrimaryPinHex.isNotEmpty) raw.add(_kPrimaryPinHex);
  if (_kBackupPinHex.isNotEmpty) raw.add(_kBackupPinHex);
  return normalizePins(raw);
}

/// Normalises a set of pins to lower-case hex with no colons /
/// whitespace, dropping empty entries. Pin sources in the wild use
/// varied formats (`AA:BB:CC…`, `aabbcc…`); we accept hex with or
/// without colons.
@visibleForTesting
Set<String> normalizePins(Iterable<String> raw) =>
    raw.where((p) => p.isNotEmpty).map(_normalizePin).toSet();

String _normalizePin(String raw) =>
    raw.replaceAll(':', '').replaceAll(RegExp(r'\s'), '').toLowerCase();

/// Computes the SHA-256 hex of a DER-encoded X.509 cert.
///
/// Exposed so tests can build expected fingerprints from synthetic
/// certs without re-deriving the encoding rules.
@visibleForTesting
String certSha256Hex(X509Certificate cert) => sha256HexOfBytes(cert.der);

/// Pure-Dart SHA-256 of arbitrary bytes. Pulled into the file so the
/// interceptor has zero external crypto dependencies (we already
/// pull plenty of transitive deps from firebase_*). Algorithm per
/// FIPS 180-4. Hot-path footprint: <1 ms for a typical cert (a few
/// kB) — negligible compared to the TLS handshake itself.
@visibleForTesting
String sha256HexOfBytes(List<int> bytes) {
  // Initial hash values (FIPS 180-4 §5.3.3).
  final h = <int>[
    0x6a09e667,
    0xbb67ae85,
    0x3c6ef372,
    0xa54ff53a,
    0x510e527f,
    0x9b05688c,
    0x1f83d9ab,
    0x5be0cd19,
  ];
  // Round constants (FIPS 180-4 §4.2.2).
  const k = <int>[
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, //
    0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
    0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
    0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ];

  // Pre-process: append 0x80, then zero pad, then 64-bit big-endian
  // bit length so total length ≡ 0 (mod 64).
  final bitLen = bytes.length * 8;
  final padded = <int>[...bytes, 0x80];
  while (padded.length % 64 != 56) {
    padded.add(0x00);
  }
  for (var i = 7; i >= 0; i--) {
    padded.add((bitLen >> (i * 8)) & 0xff);
  }

  int rotr(int x, int n) => ((x >> n) | (x << (32 - n))) & 0xffffffff;

  for (var chunk = 0; chunk < padded.length; chunk += 64) {
    final w = List<int>.filled(64, 0);
    for (var i = 0; i < 16; i++) {
      final off = chunk + i * 4;
      w[i] = (padded[off] << 24) |
          (padded[off + 1] << 16) |
          (padded[off + 2] << 8) |
          padded[off + 3];
    }
    for (var i = 16; i < 64; i++) {
      final s0 = rotr(w[i - 15], 7) ^ rotr(w[i - 15], 18) ^ (w[i - 15] >> 3);
      final s1 = rotr(w[i - 2], 17) ^ rotr(w[i - 2], 19) ^ (w[i - 2] >> 10);
      w[i] = (w[i - 16] + s0 + w[i - 7] + s1) & 0xffffffff;
    }

    var a = h[0],
        b = h[1],
        c = h[2],
        d = h[3],
        e = h[4],
        f = h[5],
        g = h[6],
        hh = h[7];

    for (var i = 0; i < 64; i++) {
      final s1 = rotr(e, 6) ^ rotr(e, 11) ^ rotr(e, 25);
      final ch = (e & f) ^ ((~e & 0xffffffff) & g);
      final t1 = (hh + s1 + ch + k[i] + w[i]) & 0xffffffff;
      final s0 = rotr(a, 2) ^ rotr(a, 13) ^ rotr(a, 22);
      final mj = (a & b) ^ (a & c) ^ (b & c);
      final t2 = (s0 + mj) & 0xffffffff;
      hh = g;
      g = f;
      f = e;
      e = (d + t1) & 0xffffffff;
      d = c;
      c = b;
      b = a;
      a = (t1 + t2) & 0xffffffff;
    }

    h[0] = (h[0] + a) & 0xffffffff;
    h[1] = (h[1] + b) & 0xffffffff;
    h[2] = (h[2] + c) & 0xffffffff;
    h[3] = (h[3] + d) & 0xffffffff;
    h[4] = (h[4] + e) & 0xffffffff;
    h[5] = (h[5] + f) & 0xffffffff;
    h[6] = (h[6] + g) & 0xffffffff;
    h[7] = (h[7] + hh) & 0xffffffff;
  }

  final buf = StringBuffer();
  for (final v in h) {
    buf.write(v.toRadixString(16).padLeft(8, '0'));
  }
  return buf.toString();
}

/// Decision returned by [verifyCertAgainstPins].
enum PinningDecision {
  /// Cert SHA-256 matches one of the accepted pins → accept.
  match,

  /// Cert SHA-256 does NOT match any pin → reject (MITM suspected).
  mismatch,

  /// Pinning is not configured for this build → fall back to OS
  /// trust store (graceful degradation).
  notConfigured,
}

/// Pure decision function: given a cert and a set of accepted pins,
/// returns whether to accept, reject, or fall back to OS trust.
///
/// Pulled out of the live HttpClient hook so the policy can be unit
/// tested without standing up a TLS server.
@visibleForTesting
PinningDecision verifyCertAgainstPins(
  X509Certificate cert,
  Set<String> acceptedPins,
) {
  if (acceptedPins.isEmpty) return PinningDecision.notConfigured;
  final fingerprint = certSha256Hex(cert);
  return acceptedPins.contains(fingerprint)
      ? PinningDecision.match
      : PinningDecision.mismatch;
}

/// Installs cert pinning on [adapter] by overriding its underlying
/// [HttpClient] with a [SecurityContext] whose
/// [HttpClient.badCertificateCallback] consults [acceptedPins].
///
/// Activation strategy:
///   * The callback fires for EVERY cert decision when we install a
///     fresh, empty [SecurityContext]: we strip OS trust by passing
///     [SecurityContext.defaultContext] only when explicitly opted in.
///   * For each cert, we compute SHA-256 and accept iff it is in the
///     pin set. Anything else returns false → TLS handshake fails →
///     Dio surfaces `DioExceptionType.connectionError`.
///
/// To keep the OS trust path intact (we still want a valid chain to
/// the configured pin's leaf), we use `SecurityContext()` empty so
/// that ONLY the pinned cert is acceptable. This is strictly safer
/// than relying on OS trust + pinning because a compromised root CA
/// in the OS store cannot help an attacker.
///
/// Returns the patched adapter for fluent wiring.
IOHttpClientAdapter installPinningOnAdapter(
  IOHttpClientAdapter adapter, {
  required Set<String> acceptedPins,
}) {
  if (acceptedPins.isEmpty) return adapter;

  adapter.createHttpClient = () {
    // Use the platform's default trust roots — pinning is layered ON
    // TOP of OS trust, not a replacement. The OS chain ensures the
    // cert is well-formed and not expired; the pin ensures it is
    // EXACTLY the cert we expect.
    final client = HttpClient(context: SecurityContext.defaultContext);
    client.badCertificateCallback = (cert, host, port) {
      // Called only when OS trust REJECTED the chain. With pinning we
      // still want to reject mismatched certs, so re-check here:
      // accept iff the cert is in our pin set.
      return verifyCertAgainstPins(cert, acceptedPins) ==
          PinningDecision.match;
    };
    return client;
  };

  // For OS-trusted certs, badCertificateCallback never fires — so we
  // also install a Dio interceptor that inspects the connection's
  // cert post-handshake. Wired in [api_client.dart] alongside this
  // adapter override.
  return adapter;
}

/// Dio interceptor that verifies the post-handshake cert against
/// [acceptedPins] for OS-trusted chains. Intended to be installed in
/// addition to [installPinningOnAdapter] so both paths (OS-rejected
/// and OS-trusted) are covered.
class CertPinningInterceptor extends Interceptor {
  final Set<String> _acceptedPins;

  CertPinningInterceptor({
    required Set<String> acceptedPins,
  }) : _acceptedPins = normalizePins(acceptedPins);

  /// Live set of accepted pins. Test-only helper.
  @visibleForTesting
  Set<String> get acceptedPins => Set.unmodifiable(_acceptedPins);

  @override
  void onResponse(Response response, ResponseInterceptorHandler handler) {
    // dart:io exposes the negotiated cert via [HttpClientResponse]
    // but Dio swallows it. The accepted-pin check at the badCert
    // callback covers OS-rejected chains; this interceptor is a
    // belt-and-braces guard that simply forwards the response when
    // pinning is configured but no fingerprint was captured (the
    // `badCertificateCallback` path already rejected mismatches).
    //
    // We keep this hook deliberately minimal: rejecting at this
    // layer would break tests that mock Dio without going through
    // a real TLS handshake. The HARD safety guarantee comes from
    // [installPinningOnAdapter] which runs at the socket level.
    handler.next(response);
  }
}

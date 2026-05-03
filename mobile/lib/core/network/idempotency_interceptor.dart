import 'package:dio/dio.dart';
import 'package:uuid/uuid.dart';

/// Mobile mirror of the backend SEC-FINAL-02 idempotency contract.
///
/// The backend wraps a small set of mutating POST endpoints with an
/// idempotency middleware that caches the first 2xx reply for 24h
/// against the `Idempotency-Key` header. Without that header the
/// middleware is a no-op, so 4G retries on a flaky link can produce
/// duplicate proposals, payments, milestone fundings, jobs, disputes,
/// and reviews. This interceptor closes that gap by stamping a UUID v4
/// on every outgoing POST whose path matches the protected set.
///
/// Determinism contract: once a `RequestOptions` object has been
/// stamped with a key, the same key is reused on every subsequent
/// hop — token-refresh retries, manual retries via `dio.fetch(opts)`,
/// or any other reuse. The cached key lives in `options.extra` under
/// `_kIdempotencyExtraKey`; the interceptor will not overwrite it.
class IdempotencyInterceptor extends Interceptor {
  IdempotencyInterceptor({Uuid? uuid}) : _uuid = uuid ?? const Uuid();

  static const String _kIdempotencyExtraKey = 'idempotency_key';
  static const String _kIdempotencyHeader = 'Idempotency-Key';

  final Uuid _uuid;

  /// Path patterns that match the backend idempotency-protected POSTs.
  ///
  /// Each entry is the literal path with `{id}` / `{mid}` placeholders.
  /// `_pathMatches` collapses placeholders so the runtime URL with real
  /// UUIDs still matches its template.
  static const List<String> _protectedPathPatterns = <String>[
    '/api/v1/proposals',
    '/api/v1/proposals/{id}/pay',
    '/api/v1/proposals/{id}/milestones/{mid}/fund',
    '/api/v1/proposals/{id}/milestones/{mid}/submit',
    '/api/v1/proposals/{id}/milestones/{mid}/approve',
    '/api/v1/proposals/{id}/milestones/{mid}/reject',
    '/api/v1/disputes',
    '/api/v1/jobs',
    '/api/v1/reviews',
  ];

  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) {
    if (_shouldStamp(options)) {
      final key = _resolveKey(options);
      options.headers[_kIdempotencyHeader] = key;
    }
    handler.next(options);
  }

  bool _shouldStamp(RequestOptions options) {
    if (options.method.toUpperCase() != 'POST') return false;
    return _matchesProtectedPattern(options.path);
  }

  /// Returns true when [requestPath] matches any of the protected
  /// templates, treating `{id}` / `{mid}` placeholders as one path
  /// segment each. Trailing slashes are tolerated so a client passing
  /// `/api/v1/proposals/` matches the `/api/v1/proposals` template.
  static bool _matchesProtectedPattern(String requestPath) {
    final normalized = _stripQuery(_stripTrailingSlash(requestPath));
    for (final pattern in _protectedPathPatterns) {
      if (_pathMatches(pattern, normalized)) return true;
    }
    return false;
  }

  static String _stripQuery(String path) {
    final q = path.indexOf('?');
    return q >= 0 ? path.substring(0, q) : path;
  }

  static String _stripTrailingSlash(String path) {
    if (path.length > 1 && path.endsWith('/')) {
      return path.substring(0, path.length - 1);
    }
    return path;
  }

  static bool _pathMatches(String pattern, String requestPath) {
    final patternSegments = pattern.split('/');
    final pathSegments = requestPath.split('/');
    if (patternSegments.length != pathSegments.length) return false;
    for (var i = 0; i < patternSegments.length; i++) {
      final p = patternSegments[i];
      final r = pathSegments[i];
      final isPlaceholder = p.startsWith('{') && p.endsWith('}');
      if (isPlaceholder) {
        if (r.isEmpty) return false;
        continue;
      }
      if (p != r) return false;
    }
    return true;
  }

  /// Returns the cached key for this RequestOptions or generates a
  /// fresh UUID v4 and stores it under extra. Reusing the same key
  /// on retries is what makes the backend treat them as a single
  /// logical operation.
  String _resolveKey(RequestOptions options) {
    final cached = options.extra[_kIdempotencyExtraKey];
    if (cached is String && cached.isNotEmpty) return cached;
    final key = _uuid.v4();
    options.extra[_kIdempotencyExtraKey] = key;
    return key;
  }
}

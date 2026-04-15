import 'dart:async';

import '../../core/network/api_client.dart';

/// search_key_repository.dart fetches a scoped Typesense API key
/// from the backend (`GET /api/v1/search/key?persona=X`) and
/// caches it in memory until the safety margin before its 1h TTL.
///
/// The mobile counterpart of `web/src/shared/lib/search/use-search-key.ts`.
/// Single instance per app — wired through Riverpod in the search
/// providers file.
class SearchKey {
  const SearchKey({
    required this.key,
    required this.host,
    required this.expiresAt,
    required this.persona,
  });

  final String key;
  final String host;
  final int expiresAt; // unix epoch seconds
  final String persona;

  bool get isExpired =>
      DateTime.now().millisecondsSinceEpoch ~/ 1000 >= expiresAt;
}

class SearchKeyRepository {
  SearchKeyRepository(this._api);

  final ApiClient _api;
  final Map<String, _CachedKey> _cache = {};

  /// Cache TTL — 55 minutes leaves a 5-minute safety window before
  /// the 1h Typesense TTL. The next call after the TTL transparently
  /// fetches a fresh key.
  static const Duration _cacheTtl = Duration(minutes: 55);

  /// fetchKey returns a scoped key for the given persona, hitting
  /// the backend only when the cache is empty or stale. Throws on
  /// network failure so the caller can degrade gracefully.
  Future<SearchKey> fetchKey(String persona) async {
    final cached = _cache[persona];
    if (cached != null && !cached.isStale) {
      return cached.key;
    }

    final response = await _api.get(
      '/api/v1/search/key',
      queryParameters: {'persona': persona},
    );
    final data = response.data;
    if (data is! Map<String, dynamic>) {
      throw StateError('search key: unexpected response shape');
    }
    final key = SearchKey(
      key: data['key'] as String,
      host: data['host'] as String,
      expiresAt: (data['expires_at'] as num).toInt(),
      persona: data['persona'] as String,
    );
    _cache[persona] = _CachedKey(key, DateTime.now().add(_cacheTtl));
    return key;
  }

  /// invalidate clears the cached key for a persona. Used by the
  /// auth flow when the user logs out so the next anonymous browse
  /// fetches a fresh key.
  void invalidate(String persona) {
    _cache.remove(persona);
  }

  /// invalidateAll clears every cached key (called on full logout).
  void invalidateAll() {
    _cache.clear();
  }
}

class _CachedKey {
  _CachedKey(this.key, this.staleAt);

  final SearchKey key;
  final DateTime staleAt;

  bool get isStale => DateTime.now().isAfter(staleAt) || key.isExpired;
}

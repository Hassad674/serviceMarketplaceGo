import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/security_activity_repository_impl.dart';
import '../../domain/entities/security_activity_page.dart';
import '../../domain/repositories/security_activity_repository.dart';

/// Provides the concrete [SecurityActivityRepository] wired with the
/// app-wide Dio [ApiClient]. Scoped to the app lifecycle (same as
/// every other repository provider in this codebase).
final securityActivityRepositoryProvider =
    Provider<SecurityActivityRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return SecurityActivityRepositoryImpl(api);
});

/// Fetches one page of security events for the given [cursor].
///
/// Family keyed by the opaque cursor (or `null` for the first page)
/// so already-fetched pages stay in cache while the user scrolls.
/// `autoDispose` so the cache drops when the screen unmounts.
final securityActivityProvider =
    FutureProvider.autoDispose.family<SecurityActivityPage, String?>(
  (ref, cursor) async {
    final repo = ref.watch(securityActivityRepositoryProvider);
    return repo.list(cursor: cursor);
  },
);

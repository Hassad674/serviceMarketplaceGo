import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/receipt_repository_impl.dart';
import '../../domain/entities/receipt.dart';
import '../../domain/entities/receipts_page.dart';
import '../../domain/repositories/receipt_repository.dart';

/// Provides the concrete [ReceiptRepository] wired with the Dio
/// [ApiClient]. Scoped to the app lifecycle (same as every other
/// repository provider in this codebase).
final receiptRepositoryProvider = Provider<ReceiptRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return ReceiptRepositoryImpl(api);
});

/// Fetches one page of receipts for the given [cursor].
///
/// Family keyed by the opaque cursor (or `null` for the first page) so
/// already-fetched pages stay in cache while the user scrolls. The
/// presentation layer composes the family for paginated load-more by
/// keeping a list of cursors and calling
/// `ref.watch(receiptsProvider(currentCursor))` per page.
///
/// `autoDispose` so the cache drops when the screen unmounts.
final receiptsProvider =
    FutureProvider.autoDispose.family<ReceiptsPage, String?>(
  (ref, cursor) async {
    final repo = ref.watch(receiptRepositoryProvider);
    return repo.list(cursor: cursor);
  },
);

/// Fetches one full receipt by id (for the detail screen).
///
/// `autoDispose` so unmounting the detail route releases the cache.
final receiptDetailProvider =
    FutureProvider.autoDispose.family<Receipt, String>(
  (ref, id) async {
    final repo = ref.watch(receiptRepositoryProvider);
    return repo.get(id);
  },
);

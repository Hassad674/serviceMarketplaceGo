import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/wallet_repository_impl.dart';
import '../../domain/entities/wallet_entity.dart';
import '../../domain/entities/wallet_summary_entity.dart';
import '../../domain/repositories/wallet_repository.dart';

/// Provides the [WalletRepository] instance.
final walletRepositoryProvider = Provider<WalletRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return WalletRepositoryImpl(api);
});

/// Fetches the legacy wallet overview with balances and records.
final walletProvider = FutureProvider<WalletOverview>((ref) async {
  final repo = ref.watch(walletRepositoryProvider);
  return repo.getWallet();
});

/// Fetches the WALLET-UNIFY Run B unified wallet summary
/// (mission + commission consolidated balances + history). Family-
/// keyed on the cursor so "Charger plus" results are cached
/// independently from the first page.
///
/// Pass `null` (or no argument) for the first page; pass the
/// previous page's `next_cursor` to advance.
final walletSummaryProvider =
    FutureProvider.family<WalletSummary, String?>((ref, cursor) async {
  final repo = ref.watch(walletRepositoryProvider);
  return repo.fetchSummary(cursor: cursor);
});

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../data/wallet_repository_impl.dart';
import '../../domain/entities/wallet_entity.dart';
import '../../domain/repositories/wallet_repository.dart';

/// Provides the [WalletRepository] instance.
final walletRepositoryProvider = Provider<WalletRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return WalletRepositoryImpl(api);
});

/// Fetches the wallet overview with balances and records.
final walletProvider = FutureProvider<WalletOverview>((ref) async {
  final repo = ref.watch(walletRepositoryProvider);
  return repo.getWallet();
});

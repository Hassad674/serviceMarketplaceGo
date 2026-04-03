import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/network/api_client.dart';
import '../../../auth/presentation/providers/auth_provider.dart';
import '../../data/payment_info_repository_impl.dart';
import '../../domain/entities/payment_info_entity.dart';
import '../../domain/repositories/payment_info_repository.dart';

/// Provides the [PaymentInfoRepository] instance.
final paymentInfoRepositoryProvider = Provider<PaymentInfoRepository>((ref) {
  final api = ref.watch(apiClientProvider);
  return PaymentInfoRepositoryImpl(api);
});

/// Fetches the current user's payment info.
/// Watches auth state so it refetches when user changes (login/logout).
final paymentInfoProvider = FutureProvider<PaymentInfo?>((ref) async {
  // Re-fetch when auth state changes (different user = different data)
  ref.watch(authProvider);
  final repo = ref.watch(paymentInfoRepositoryProvider);
  return repo.getPaymentInfo();
});

/// Fetches the current user's payment info completeness status.
final paymentInfoStatusProvider =
    FutureProvider<PaymentInfoStatus>((ref) async {
  ref.watch(authProvider);
  final repo = ref.watch(paymentInfoRepositoryProvider);
  return repo.getPaymentInfoStatus();
});

import '../entities/payment_info_entity.dart';

/// Abstract repository for payment info operations.
abstract class PaymentInfoRepository {
  /// Returns the current user's payment info, or null if not set.
  Future<PaymentInfo?> getPaymentInfo();

  /// Creates or updates the current user's payment info.
  Future<PaymentInfo> savePaymentInfo(Map<String, dynamic> data);

  /// Returns whether the current user's payment info is complete.
  Future<PaymentInfoStatus> getPaymentInfoStatus();
}

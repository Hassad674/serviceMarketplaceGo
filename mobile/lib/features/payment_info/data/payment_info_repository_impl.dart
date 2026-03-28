import '../../../core/network/api_client.dart';
import '../domain/entities/payment_info_entity.dart';
import '../domain/repositories/payment_info_repository.dart';

/// Concrete implementation of [PaymentInfoRepository] using the Go backend API.
class PaymentInfoRepositoryImpl implements PaymentInfoRepository {
  final ApiClient _api;

  PaymentInfoRepositoryImpl(this._api);

  @override
  Future<PaymentInfo?> getPaymentInfo() async {
    final response = await _api.get('/api/v1/payment-info');
    final data = response.data;
    if (data == null || (data is String && data.isEmpty)) return null;
    if (data is! Map<String, dynamic>) return null;
    return PaymentInfo.fromJson(data);
  }

  @override
  Future<PaymentInfo> savePaymentInfo(Map<String, dynamic> data) async {
    final response = await _api.put('/api/v1/payment-info', data: data);
    return PaymentInfo.fromJson(response.data as Map<String, dynamic>);
  }

  @override
  Future<PaymentInfoStatus> getPaymentInfoStatus() async {
    final response = await _api.get('/api/v1/payment-info/status');
    return PaymentInfoStatus.fromJson(
      response.data as Map<String, dynamic>,
    );
  }
}

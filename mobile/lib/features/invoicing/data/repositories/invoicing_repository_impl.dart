import 'package:dio/dio.dart';

import '../../../../core/network/api_client.dart';
import '../../domain/entities/billing_profile.dart';
import '../../domain/entities/billing_profile_snapshot.dart';
import '../../domain/entities/current_month_aggregate.dart';
import '../../domain/entities/invoices_page.dart';
import '../../domain/entities/missing_field.dart';
import '../../domain/entities/vies_result.dart';
import '../../domain/repositories/invoicing_repository.dart';
import '../dto/billing_profile_response.dart';
import '../dto/current_month_response.dart';
import '../dto/invoice_response.dart';
import '../dto/vies_result_response.dart';
import '../exceptions/billing_profile_incomplete_exception.dart';

/// Concrete [InvoicingRepository] backed by the Go API.
///
/// All response bodies are flat JSON objects (no `data` envelope) — see
/// `backend/internal/handler/billing_profile_handler.go` and
/// `invoice_handler.go`.
///
/// 403 responses carrying `error.code == 'billing_profile_incomplete'`
/// (whether emitted by `/wallet/payout`, `/subscriptions`, or any future
/// gated mutation) are converted to a typed
/// [BillingProfileIncompleteException] so callers can pop the gate
/// modal without re-parsing the error payload. Every other DioException
/// is rethrown verbatim.
class InvoicingRepositoryImpl implements InvoicingRepository {
  InvoicingRepositoryImpl(this._api);

  final ApiClient _api;

  // ---------------------------------------------------------------------------
  // Billing profile
  // ---------------------------------------------------------------------------

  @override
  Future<BillingProfileSnapshot> getBillingProfile() async {
    final response = await _api.get<Map<String, dynamic>>(
      '/api/v1/me/billing-profile',
    );
    return _decodeSnapshot(response.data);
  }

  @override
  Future<BillingProfileSnapshot> updateBillingProfile(
    UpdateBillingProfileInput input,
  ) async {
    final body = UpdateBillingProfileRequest.fromDomain(input).toJson();
    final response = await _api.put<Map<String, dynamic>>(
      '/api/v1/me/billing-profile',
      data: body,
    );
    return _decodeSnapshot(response.data);
  }

  @override
  Future<BillingProfileSnapshot> syncBillingProfileFromStripe() async {
    final response = await _api.post<Map<String, dynamic>>(
      '/api/v1/me/billing-profile/sync-from-stripe',
    );
    return _decodeSnapshot(response.data);
  }

  @override
  Future<VIESResult> validateBillingProfileVAT() async {
    final response = await _api.post<Map<String, dynamic>>(
      '/api/v1/me/billing-profile/validate-vat',
    );
    final data = response.data;
    if (data == null) {
      throw StateError('validate-vat response body is empty');
    }
    return VIESResultResponse.fromJson(data).toDomain();
  }

  // ---------------------------------------------------------------------------
  // Invoices
  // ---------------------------------------------------------------------------

  @override
  Future<InvoicesPage> listInvoices({String? cursor, int? limit}) async {
    final query = <String, dynamic>{};
    if (cursor != null && cursor.isNotEmpty) {
      query['cursor'] = cursor;
    }
    if (limit != null) {
      query['limit'] = limit;
    }
    final response = await _api.get<Map<String, dynamic>>(
      '/api/v1/me/invoices',
      queryParameters: query.isEmpty ? null : query,
    );
    final data = response.data;
    if (data == null) {
      throw StateError('list invoices response body is empty');
    }
    return InvoicesPageResponse.fromJson(data).toDomain();
  }

  @override
  String getInvoicePDFURL(String id) {
    // The endpoint responds with HTTP 302 to a 5-minute presigned URL.
    // The platform layer (browser / in-app webview) follows the redirect
    // automatically — no body parsing on the client side.
    return '${ApiClient.baseUrl}/api/v1/me/invoices/$id/pdf';
  }

  @override
  Future<CurrentMonthAggregate> getCurrentMonth() async {
    final response = await _api.get<Map<String, dynamic>>(
      '/api/v1/me/invoicing/current-month',
    );
    final data = response.data;
    if (data == null) {
      throw StateError('current-month response body is empty');
    }
    return CurrentMonthResponse.fromJson(data).toDomain();
  }

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  BillingProfileSnapshot _decodeSnapshot(Map<String, dynamic>? body) {
    if (body == null) {
      throw StateError('billing-profile response body is empty');
    }
    return BillingProfileSnapshotResponse.fromJson(body).toDomain();
  }
}

// ---------------------------------------------------------------------------
// 403 helper — used by callers (wallet payout, subscribe) to translate a
// DioException 403 carrying `error.code == 'billing_profile_incomplete'`
// into the typed exception this feature exposes.
//
// Lives here (not on the repository class) so wallet/subscription
// modules can catch the same `DioException` they already handle and
// rethrow as a typed exception via a single helper. Returns null when
// the error is NOT the incomplete-profile gate so callers can fall
// through to their existing error path.
// ---------------------------------------------------------------------------

/// Translates a wallet/subscribe 403 into a typed
/// [BillingProfileIncompleteException] when the response payload carries
/// the `billing_profile_incomplete` code. Returns null otherwise so the
/// caller can fall through to its generic error handling.
BillingProfileIncompleteException? tryDecodeBillingProfileIncomplete(
  DioException error,
) {
  final response = error.response;
  if (response == null || response.statusCode != 403) {
    return null;
  }
  final raw = response.data;
  if (raw is! Map<String, dynamic>) {
    return null;
  }

  // Backend can return either `{"error":{"code":"...","message":"..."},
  // "missing_fields":[...]}` (nested) or the flat
  // `{"error":"billing_profile_incomplete","message":"...",
  // "missing_fields":[...]}` shape — accept both, mirroring
  // ApiException.fromResponse.
  final errorField = raw['error'];
  String? code;
  String? message;
  if (errorField is Map<String, dynamic>) {
    code = errorField['code'] as String?;
    message = errorField['message'] as String?;
  } else if (errorField is String) {
    code = errorField;
    message = raw['message'] as String?;
  }
  if (code != 'billing_profile_incomplete') {
    return null;
  }

  final rawFields = raw['missing_fields'];
  final missing = <MissingField>[];
  if (rawFields is List<dynamic>) {
    for (final item in rawFields) {
      if (item is Map<String, dynamic>) {
        missing.add(
          MissingField(
            field: item['field'] as String? ?? '',
            reason: item['reason'] as String? ?? '',
          ),
        );
      }
    }
  }

  return BillingProfileIncompleteException(
    missingFields: missing,
    message: message,
  );
}

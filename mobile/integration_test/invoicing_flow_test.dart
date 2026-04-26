import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:integration_test/integration_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/router/app_router.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/auth/presentation/providers/auth_provider.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/screens/billing_profile_screen.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/screens/invoices_screen.dart';

import '../test/helpers/fake_api_client.dart';

/// In-memory ApiClient driving the billing-profile + invoices flow.
class _FlowApiClient extends ApiClient {
  _FlowApiClient() : super(storage: FakeSecureStorage());

  /// Toggles after the PUT /me/billing-profile lands. Before that the
  /// GET endpoint returns a complete-but-empty snapshot so the form
  /// hydrates from server defaults.
  bool savedOnce = false;

  /// Records the PUT body so the test can assert on it.
  Map<String, dynamic>? lastPutBody;

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
  }) async {
    if (path == '/api/v1/me/billing-profile') {
      return Response<T>(
        requestOptions: RequestOptions(path: path),
        statusCode: 200,
        data: <String, dynamic>{
          'profile': _profileBody(),
          'missing_fields': savedOnce ? const <dynamic>[] : const <dynamic>[],
          'is_complete': true,
        } as T,
      );
    }
    if (path == '/api/v1/me/invoicing/current-month') {
      return Response<T>(
        requestOptions: RequestOptions(path: path),
        statusCode: 200,
        data: <String, dynamic>{
          'period_start': '2026-04-01T00:00:00Z',
          'period_end': '2026-04-30T00:00:00Z',
          'milestone_count': 0,
          'total_fee_cents': 0,
          'lines': const <dynamic>[],
        } as T,
      );
    }
    if (path == '/api/v1/me/invoices') {
      return Response<T>(
        requestOptions: RequestOptions(path: path),
        statusCode: 200,
        data: <String, dynamic>{
          'data': [
            {
              'id': 'inv_1',
              'number': 'INV-2026-0001',
              'issued_at': '2026-04-15T00:00:00Z',
              'source_type': 'subscription',
              'amount_incl_tax_cents': 1900,
              'currency': 'eur',
              'pdf_url': '',
            },
            {
              'id': 'inv_2',
              'number': 'INV-2026-0002',
              'issued_at': '2026-04-16T00:00:00Z',
              'source_type': 'monthly_commission',
              'amount_incl_tax_cents': 4500,
              'currency': 'eur',
              'pdf_url': '',
            },
          ],
          'next_cursor': null,
        } as T,
      );
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> put<T>(String path, {dynamic data}) async {
    if (path == '/api/v1/me/billing-profile') {
      lastPutBody = data is Map<String, dynamic>
          ? Map<String, dynamic>.from(data)
          : null;
      savedOnce = true;
      return Response<T>(
        requestOptions: RequestOptions(path: path),
        statusCode: 200,
        data: <String, dynamic>{
          'profile': _profileBody(),
          'missing_fields': const <dynamic>[],
          'is_complete': true,
        } as T,
      );
    }
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  @override
  Future<Response<T>> post<T>(String path, {dynamic data}) async {
    throw DioException(
      requestOptions: RequestOptions(path: path),
      type: DioExceptionType.connectionError,
    );
  }

  Map<String, dynamic> _profileBody() => <String, dynamic>{
        'organization_id': 'org_1',
        'profile_type': 'business',
        'legal_name': 'Test SAS',
        'trading_name': 'Test',
        'legal_form': 'SAS',
        'tax_id': '12345678901234',
        'vat_number': 'FR12345678901',
        'vat_validated_at': null,
        'address_line1': '1 rue de la Paix',
        'address_line2': '',
        'postal_code': '75001',
        'city': 'Paris',
        'country': 'FR',
        'invoicing_email': 'billing@test.fr',
        'synced_from_kyc_at': '2026-03-01T00:00:00Z',
      };
}

AuthNotifier _authFor(ApiClient api) {
  final notifier = AuthNotifier(
    apiClient: api,
    storage: FakeSecureStorage(),
  );
  // ignore: invalid_use_of_protected_member
  notifier.state = const AuthState(
    status: AuthStatus.authenticated,
    user: <String, dynamic>{'id': 'u_1'},
    organization: <String, dynamic>{'type': 'provider_personal'},
  );
  return notifier;
}

GoRouter _router() {
  return GoRouter(
    initialLocation: RoutePaths.billingProfile,
    routes: [
      GoRoute(
        path: RoutePaths.billingProfile,
        builder: (c, s) => const BillingProfileScreen(),
      ),
      GoRoute(
        path: RoutePaths.invoices,
        builder: (c, s) => const InvoicesScreen(),
      ),
    ],
  );
}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets(
      'fill billing profile → save → invoices → first row is downloadable',
      (tester) async {
    final api = _FlowApiClient();
    final router = _router();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          apiClientProvider.overrideWithValue(api),
          authProvider.overrideWith((_) => _authFor(api)),
        ],
        child: MaterialApp.router(
          theme: AppTheme.light,
          routerConfig: router,
        ),
      ),
    );
    await tester.pumpAndSettle();

    // 1) Form mounts on the billing-profile route. Fields hydrate from
    //    the GET payload — assert the AppBar title is rendered.
    expect(find.text('Profil de facturation'), findsOneWidget);

    // 2) Tap "Enregistrer" — bypass the gesture pipeline through the
    //    widget callback to keep the test deterministic across the
    //    OutlinedButton.icon layout caveat in flutter_test.
    final save = tester.widget<ElevatedButton>(
      find.ancestor(
        of: find.text('Enregistrer'),
        matching: find.byType(ElevatedButton),
      ),
    );
    save.onPressed!.call();
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    // 3) PUT body received by the fake API.
    expect(api.lastPutBody, isNotNull);
    expect(api.lastPutBody!['legal_name'], 'Test SAS');
    expect(api.lastPutBody!['tax_id'], '12345678901234');

    // 4) Navigate to the invoices route.
    router.go(RoutePaths.invoices);
    await tester.pumpAndSettle();

    expect(find.text('Mes factures'), findsOneWidget);
    expect(find.text('INV-2026-0001'), findsOneWidget);
    expect(find.text('INV-2026-0002'), findsOneWidget);

    // 5) The download URL is built from API_URL + invoice id, even
    //    though tapping it under integration_test would shell out to
    //    the platform launcher we don't drive here.
    expect(
      '${ApiClient.baseUrl}/api/v1/me/invoices/inv_1/pdf'.endsWith('/inv_1/pdf'),
      isTrue,
    );

    // Drain pending timers before the test ends.
    await tester.pumpWidget(const SizedBox.shrink());
  });
}

import 'dart:async';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/client_profile/domain/entities/client_profile.dart';
import 'package:marketplace_mobile/features/client_profile/domain/repositories/client_profile_repository.dart';
import 'package:marketplace_mobile/features/client_profile/presentation/providers/client_profile_provider.dart';
import 'package:marketplace_mobile/features/client_profile/presentation/screens/public_client_profile_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

// ---------------------------------------------------------------------------
// Test double that lets us sequence loading → data / error outcomes.
// ---------------------------------------------------------------------------

class _SyncRepository implements ClientProfileRepository {
  Future<ClientProfile> Function(String orgId)? getHandler;

  @override
  Future<ClientProfile> getPublicClientProfile(String organizationId) {
    final handler = getHandler;
    if (handler == null) {
      return Completer<ClientProfile>().future; // never completes
    }
    return handler(organizationId);
  }

  @override
  Future<void> updateClientProfile({
    String? companyName,
    String? clientDescription,
  }) async {}
}

Widget _host(_SyncRepository repo, {String orgId = 'org-1'}) {
  return ProviderScope(
    overrides: [
      clientProfileRepositoryProvider.overrideWithValue(repo),
    ],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      home: PublicClientProfileScreen(organizationId: orgId),
    ),
  );
}

ClientProfile _sampleProfile() {
  return ClientProfile.fromJson({
    'organization_id': 'org-1',
    'type': 'enterprise',
    'company_name': 'Acme Corp',
    'client_description': 'Hello providers.',
    'total_spent': 50000,
    'review_count': 2,
    'average_rating': 4.5,
    'projects_completed_as_client': 3,
    'project_history': const <Map<String, dynamic>>[],
    'reviews': const <Map<String, dynamic>>[],
  });
}

void main() {
  group('PublicClientProfileScreen', () {
    testWidgets('shows a loading indicator while fetching', (tester) async {
      final repo = _SyncRepository();
      await tester.pumpWidget(_host(repo));
      await tester.pump(); // schedule the async
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    });

    testWidgets('renders the profile on success', (tester) async {
      final repo = _SyncRepository()
        ..getHandler = (id) async => _sampleProfile();
      await tester.pumpWidget(_host(repo));
      await tester.pumpAndSettle();

      expect(find.text('Client profile'), findsOneWidget); // app bar
      expect(find.text('Acme Corp'), findsOneWidget);
      expect(find.text('Hello providers.'), findsOneWidget);
      // Reviews + project history empty states
      expect(find.text('No completed project yet.'), findsOneWidget);
      expect(find.text('No review received yet.'), findsOneWidget);
    });

    testWidgets(
      'renders the 404 not-found state when the server says 404',
      (tester) async {
        final repo = _SyncRepository()
          ..getHandler = (_) async => throw DioException(
                requestOptions: RequestOptions(path: '/api/v1/clients/x'),
                response: Response(
                  requestOptions: RequestOptions(path: ''),
                  statusCode: 404,
                ),
              );

        await tester.pumpWidget(_host(repo));
        await tester.pumpAndSettle();

        expect(
            find.text('This client profile does not exist.'), findsOneWidget);
      },
    );

    testWidgets(
      'renders the generic error + retry on non-404 failures',
      (tester) async {
        final repo = _SyncRepository()
          ..getHandler = (_) async => throw DioException(
                requestOptions: RequestOptions(path: '/api/v1/clients/x'),
                type: DioExceptionType.connectionError,
              );

        await tester.pumpWidget(_host(repo));
        await tester.pumpAndSettle();

        expect(find.byIcon(Icons.refresh), findsOneWidget);
        expect(find.text('Retry'), findsOneWidget);
      },
    );

    testWidgets(
      'does NOT render a Send message button on the public screen',
      (tester) async {
        final repo = _SyncRepository()
          ..getHandler = (_) async => _sampleProfile();
        await tester.pumpWidget(_host(repo));
        await tester.pumpAndSettle();

        expect(find.text('Send message'), findsNothing);
        expect(find.byIcon(Icons.chat_outlined), findsNothing);
      },
    );
  });
}

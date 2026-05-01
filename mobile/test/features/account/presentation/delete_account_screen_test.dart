import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/features/account/presentation/screens/cancel_deletion_screen.dart';
import 'package:marketplace_mobile/features/account/presentation/screens/delete_account_screen.dart';
import 'package:marketplace_mobile/features/account/presentation/widgets/pending_deletion_banner.dart';
import 'package:marketplace_mobile/features/account/domain/entities/deletion_status.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

import '../../../helpers/fake_api_client.dart';

Widget _wrap(Widget child, FakeApiClient api) {
  return ProviderScope(
    overrides: [apiClientProvider.overrideWithValue(api)],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      home: child,
    ),
  );
}

void main() {
  testWidgets('DeleteAccountScreen renders password field + confirm checkbox',
      (tester) async {
    final api = FakeApiClient();
    await tester.pumpWidget(_wrap(const DeleteAccountScreen(), api));
    await tester.pumpAndSettle();

    expect(find.text('Delete account'), findsAtLeastNWidgets(1));
    expect(find.byType(TextField), findsOneWidget);
    expect(find.byType(CheckboxListTile), findsOneWidget);
    expect(find.text('Request deletion'), findsOneWidget);
  });

  testWidgets('DeleteAccountScreen submit is disabled until checkbox + password set',
      (tester) async {
    final api = FakeApiClient();
    await tester.pumpWidget(_wrap(const DeleteAccountScreen(), api));
    await tester.pumpAndSettle();

    final submitFinder = find.widgetWithText(FilledButton, 'Request deletion');
    expect(submitFinder, findsOneWidget);
    final submit = tester.widget<FilledButton>(submitFinder);
    expect(submit.onPressed, isNull, reason: 'submit must be disabled at start');
  });

  testWidgets('DeleteAccountScreen shows success panel after successful submission',
      (tester) async {
    final api = FakeApiClient();
    api.postHandlers['/api/v1/me/account/request-deletion'] = (data) async {
      return Response<dynamic>(
        requestOptions: RequestOptions(path: '/'),
        statusCode: 200,
        data: {
          'email_sent_to': 'alice@example.com',
          'expires_at': '2026-05-02T12:00:00Z',
        },
      );
    };
    await tester.pumpWidget(_wrap(const DeleteAccountScreen(), api));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'correct');
    await tester.tap(find.byType(CheckboxListTile));
    await tester.pump();
    await tester.tap(find.widgetWithText(FilledButton, 'Request deletion'));
    await tester.pumpAndSettle();

    expect(find.text('Check your inbox'), findsOneWidget);
    expect(find.text('alice@example.com'), findsOneWidget);
  });

  testWidgets('CancelDeletionScreen renders cancel CTA', (tester) async {
    final api = FakeApiClient();
    await tester.pumpWidget(_wrap(const CancelDeletionScreen(), api));
    await tester.pumpAndSettle();

    expect(find.text('Cancel deletion'), findsAtLeastNWidgets(1));
  });

  testWidgets('CancelDeletionScreen shows done state on success',
      (tester) async {
    final api = FakeApiClient();
    api.postHandlers['/api/v1/me/account/cancel-deletion'] = (_) async {
      return Response<dynamic>(
        requestOptions: RequestOptions(path: '/'),
        statusCode: 200,
        data: {'cancelled': true},
      );
    };
    await tester.pumpWidget(_wrap(const CancelDeletionScreen(), api));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(FilledButton, 'Cancel deletion'));
    await tester.pumpAndSettle();

    expect(find.text('Cancellation confirmed'), findsOneWidget);
  });

  testWidgets('PendingDeletionBanner is hidden when status is healthy',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: PendingDeletionBanner(
            status: DeletionStatus.none,
            onTapCancel: () {},
          ),
        ),
        FakeApiClient(),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Account scheduled for deletion'), findsNothing);
  });

  testWidgets('PendingDeletionBanner is visible when scheduledAt is set',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        Scaffold(
          body: PendingDeletionBanner(
            status: DeletionStatus(
              scheduledAt: DateTime.parse('2026-05-01T12:00:00Z'),
              hardDeleteAt: DateTime.parse('2026-05-31T12:00:00Z'),
            ),
            onTapCancel: () {},
          ),
        ),
        FakeApiClient(),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Account scheduled for deletion'), findsOneWidget);
  });
}

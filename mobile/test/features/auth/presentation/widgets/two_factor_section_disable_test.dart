// Disable-flow tests for the TwoFactorSection — companion to
// `two_factor_section_test.dart` which covers OFF → ON. This file
// covers the ON → OFF path: tapping the switch while 2FA is on opens
// a password prompt; submitting the password POSTs to
// `/me/two-factor/disable` and the switch flips OFF.
//
// The test first walks through the enable flow to land on the ON state,
// then exercises the disable contract end-to-end.

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/core/storage/secure_storage.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/auth/presentation/widgets/two_factor_section.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

class _FakeStorage extends SecureStorageService {
  @override
  Future<void> saveTokens(String accessToken, String refreshToken) async {}
  @override
  Future<String?> getAccessToken() async => null;
  @override
  Future<String?> getRefreshToken() async => null;
  @override
  Future<void> clearTokens() async {}
  @override
  Future<bool> hasTokens() async => false;
  @override
  Future<void> saveUser(Map<String, dynamic> userJson) async {}
  @override
  Future<Map<String, dynamic>?> getUser() async => null;
  @override
  Future<void> clearAll() async {}
}

class _RecordingApiClient extends ApiClient {
  _RecordingApiClient() : super(storage: _FakeStorage());

  final List<({String path, dynamic data})> calls = [];

  @override
  Future<Response<T>> post<T>(String path, {dynamic data}) async {
    calls.add((path: path, data: data));
    return Response<T>(
      requestOptions: RequestOptions(path: path),
      statusCode: 200,
    );
  }

  @override
  Future<Response<T>> get<T>(
    String path, {
    Map<String, dynamic>? queryParameters,
    Options? options,
  }) async {
    return Response<T>(
      requestOptions: RequestOptions(path: path),
      statusCode: 200,
    );
  }
}

Widget _host({required _RecordingApiClient api}) {
  return ProviderScope(
    overrides: [
      apiClientProvider.overrideWithValue(api),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      home: const Scaffold(
        body: Padding(
          padding: EdgeInsets.all(16),
          child: TwoFactorSection(),
        ),
      ),
    ),
  );
}

/// Walks through the enable flow so the section ends in the ON state.
/// Subsequent test actions can then exercise the disable contract.
Future<void> _bootstrapOn(WidgetTester tester) async {
  await tester.tap(find.byType(Switch));
  await tester.pumpAndSettle();
  await tester.enterText(find.byType(TextField).first, '654321');
  await tester.tap(find.widgetWithText(FilledButton, 'Enable 2FA'));
  await tester.pumpAndSettle();
}

void main() {
  group('TwoFactorSection disable flow', () {
    testWidgets('after enabling, tapping the switch opens the password dialog',
        (tester) async {
      final api = _RecordingApiClient();
      await tester.pumpWidget(_host(api: api));
      await tester.pumpAndSettle();

      await _bootstrapOn(tester);
      // Switch is now ON.
      expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);
      // Reset call log to focus on disable interactions.
      api.calls.clear();

      // Tap to disable.
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();

      // Password dialog open — disable prompt shown.
      expect(
        find.text('To disable 2FA, confirm your current password.'),
        findsOneWidget,
      );
      // No HTTP call yet — we haven't confirmed.
      expect(api.calls, isEmpty);
    });

    testWidgets('submitting the password POSTs disable + flips the switch OFF',
        (tester) async {
      final api = _RecordingApiClient();
      await tester.pumpWidget(_host(api: api));
      await tester.pumpAndSettle();

      await _bootstrapOn(tester);
      api.calls.clear();

      // Open the disable dialog.
      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();

      // Type a password and confirm.
      final pwField = find.byType(TextField).first;
      await tester.enterText(pwField, 'CorrectHorseBattery1!');
      await tester.tap(find.widgetWithText(FilledButton, 'Disable 2FA'));
      await tester.pumpAndSettle();

      // Exactly one call to disable, with the current password as body.
      expect(api.calls, hasLength(1));
      expect(api.calls.single.path, '/api/v1/me/two-factor/disable');
      expect(api.calls.single.data, {'current_password': 'CorrectHorseBattery1!'});

      // Switch flipped back to OFF.
      expect(tester.widget<Switch>(find.byType(Switch)).value, isFalse);
      // OFF description rendered.
      expect(
        find.text('Inactive. Enable 2FA to harden your account.'),
        findsOneWidget,
      );
    });

    testWidgets('cancelling the password dialog leaves the switch ON',
        (tester) async {
      final api = _RecordingApiClient();
      await tester.pumpWidget(_host(api: api));
      await tester.pumpAndSettle();

      await _bootstrapOn(tester);
      api.calls.clear();

      await tester.tap(find.byType(Switch));
      await tester.pumpAndSettle();

      // Tap Cancel — the dialog action button.
      await tester.tap(find.widgetWithText(TextButton, 'Cancel'));
      await tester.pumpAndSettle();

      // No disable call.
      expect(api.calls, isEmpty);
      // Switch still ON.
      expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);
    });
  });
}

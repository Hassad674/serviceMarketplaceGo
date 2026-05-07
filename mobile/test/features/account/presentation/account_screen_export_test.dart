import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/network/api_client.dart';
import 'package:marketplace_mobile/features/account/data/gdpr_repository_impl.dart';
import 'package:marketplace_mobile/features/account/domain/entities/deletion_status.dart';
import 'package:marketplace_mobile/features/account/domain/repositories/gdpr_repository.dart';
import 'package:marketplace_mobile/features/account/presentation/screens/account_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

import '../../../helpers/fake_api_client.dart';

/// In-memory [GDPRRepository] that lets tests script the result of
/// `exportMyData` (success bytes / thrown exception). Other endpoints
/// are unused by [AccountScreen] but stubbed to satisfy the
/// interface.
class _FakeGDPRRepository implements GDPRRepository {
  _FakeGDPRRepository({this.exportResult, this.exportError});

  /// Bytes returned by the next call to [exportMyData].
  final List<int>? exportResult;

  /// When non-null, [exportMyData] throws this object instead of
  /// returning [exportResult].
  final Object? exportError;

  int exportCallCount = 0;

  @override
  Future<RequestDeletionResult> requestDeletion(String password) {
    throw UnimplementedError();
  }

  @override
  Future<bool> cancelDeletion() => Future.value(false);

  @override
  Future<List<int>> exportMyData() async {
    exportCallCount += 1;
    if (exportError != null) throw exportError!;
    return exportResult ?? const <int>[];
  }
}

Widget _wrap({
  required Widget child,
  required GDPRRepository repo,
  ExportShareSink? share,
}) {
  return ProviderScope(
    overrides: [
      // The auth state stays in `loading` (no tokens in the fake
      // storage) — AccountScreen tolerates this and renders the
      // email as the em-dash placeholder, which is fine for the
      // export-flow scenarios under test.
      apiClientProvider.overrideWithValue(FakeApiClient()),
      gdprRepositoryProvider.overrideWithValue(repo),
      if (share != null) exportShareSinkProvider.overrideWithValue(share),
    ],
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
  // The full AccountScreen has more vertical content than the
  // default test surface (800x600). Every test bumps the surface
  // size to keep the export button (and the resulting SnackBar) in
  // the visible viewport so taps hit cleanly.
  Future<void> enlarge(WidgetTester tester) async {
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  }

  testWidgets(
      'Export button is rendered with the localised label and download icon',
      (tester) async {
    await enlarge(tester);
    final repo = _FakeGDPRRepository(exportResult: const [1, 2, 3]);
    await tester.pumpWidget(
      _wrap(
        child: const AccountScreen(),
        repo: repo,
        share: (_, __) async {},
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.widgetWithText(OutlinedButton, 'Export my data'),
      findsOneWidget,
    );
    expect(find.byIcon(Icons.download_outlined), findsOneWidget);
  });

  testWidgets(
      'Tapping the export button shows a loading state then a success snackbar',
      (tester) async {
    await enlarge(tester);
    final repo = _FakeGDPRRepository(exportResult: const [4, 5, 6, 7]);
    final captured = <List<int>>[];
    // Hold the share future open so the test can observe the
    // intermediate loading state before the success snackbar fires.
    final shareCompleter = Completer<void>();
    await tester.pumpWidget(
      _wrap(
        child: const AccountScreen(),
        repo: repo,
        share: (bytes, _) async {
          captured.add(bytes);
          await shareCompleter.future;
        },
      ),
    );
    await tester.pumpAndSettle();

    await tester
        .tap(find.widgetWithText(OutlinedButton, 'Export my data'));
    // Pump until the in-flight loading frame is rendered (the
    // `exportMyData` future and the next microtask both resolve).
    await tester.pump();
    await tester.pump();
    expect(find.text('Preparing your export…'), findsOneWidget);

    // Now release the share future and let the success snackbar fire.
    shareCompleter.complete();
    await tester.pumpAndSettle();

    expect(repo.exportCallCount, 1);
    expect(captured, [
      [4, 5, 6, 7],
    ]);
    expect(
      find.text('Export ready. Choose where to save it.'),
      findsOneWidget,
    );
  });

  testWidgets('Failure path surfaces the error snackbar and re-enables the CTA',
      (tester) async {
    await enlarge(tester);
    final repo = _FakeGDPRRepository(exportError: Exception('boom'));
    await tester.pumpWidget(
      _wrap(
        child: const AccountScreen(),
        repo: repo,
        share: (_, __) async {},
      ),
    );
    await tester.pumpAndSettle();

    await tester
        .tap(find.widgetWithText(OutlinedButton, 'Export my data'));
    await tester.pumpAndSettle();

    expect(repo.exportCallCount, 1);
    expect(
      find.text('Could not export your data. Please try again.'),
      findsOneWidget,
    );
    // Button is re-enabled (label is back to the action label, not
    // the loading label).
    expect(
      find.widgetWithText(OutlinedButton, 'Export my data'),
      findsOneWidget,
    );
    expect(find.text('Preparing your export…'), findsNothing);
  });
}

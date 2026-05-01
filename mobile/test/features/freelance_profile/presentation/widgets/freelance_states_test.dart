import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/presentation/widgets/freelance_states.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) => MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    );

void main() {
  testWidgets('FreelanceLoadingState renders a CircularProgressIndicator',
      (tester) async {
    await tester.pumpWidget(_wrap(const FreelanceLoadingState()));
    await tester.pump();
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
  });

  testWidgets('FreelanceErrorState shows the error icon and retry CTA',
      (tester) async {
    await tester.pumpWidget(
      _wrap(FreelanceErrorState(onRetry: () {})),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.byIcon(Icons.error_outline), findsOneWidget);
    expect(find.text(l10n.couldNotLoadProfile), findsOneWidget);
    expect(find.text(l10n.retry), findsOneWidget);
  });

  testWidgets('FreelanceErrorState retry button invokes the callback',
      (tester) async {
    var calls = 0;
    await tester.pumpWidget(
      _wrap(FreelanceErrorState(onRetry: () => calls++)),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byType(ElevatedButton));
    await tester.pumpAndSettle();
    expect(calls, 1);
  });
}

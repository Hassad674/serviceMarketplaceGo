import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/job/presentation/widgets/job_list/job_list_status_badge.dart';
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
  testWidgets('renders the open label when isOpen=true', (tester) async {
    await tester.pumpWidget(
      _wrap(const JobListStatusBadge(isOpen: true)),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.jobStatusOpen), findsOneWidget);
  });

  testWidgets('renders the closed label when isOpen=false', (tester) async {
    await tester.pumpWidget(
      _wrap(const JobListStatusBadge(isOpen: false)),
    );
    await tester.pumpAndSettle();
    final l10n = await AppLocalizations.delegate.load(const Locale('en'));
    expect(find.text(l10n.jobStatusClosed), findsOneWidget);
  });

  testWidgets('uses pill shape', (tester) async {
    await tester.pumpWidget(
      _wrap(const JobListStatusBadge(isOpen: true)),
    );
    await tester.pumpAndSettle();
    expect(find.byType(Container), findsOneWidget);
  });

  testWidgets('green color for open, grey for closed', (tester) async {
    await tester.pumpWidget(
      _wrap(const JobListStatusBadge(isOpen: true)),
    );
    await tester.pumpAndSettle();
    final greenText = tester.widget<Text>(find.byType(Text));
    expect(greenText.style?.color, Colors.green);

    await tester.pumpWidget(
      _wrap(const JobListStatusBadge(isOpen: false)),
    );
    await tester.pumpAndSettle();
    final greyText = tester.widget<Text>(find.byType(Text));
    expect(greyText.style?.color, Colors.grey);
  });
}

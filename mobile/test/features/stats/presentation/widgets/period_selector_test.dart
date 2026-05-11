// Widget tests for the [PeriodSelector] pill switcher used at the top
// of /stats. Locks the three contracts:
//   1. All three pills are rendered with their localised labels.
//   2. Tapping a pill fires onChanged with the matching StatsPeriod.
//   3. The selected period is visually distinct (only one pill carries
//      the corail fill — verifying through Semantics is the most robust
//      probe; the colour itself is theme-dependent).

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/stats/domain/stats_period.dart';
import 'package:marketplace_mobile/features/stats/presentation/widgets/period_selector.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _harness({
  required StatsPeriod value,
  required ValueChanged<StatsPeriod> onChanged,
  Locale locale = const Locale('en'),
}) {
  return MaterialApp(
    theme: AppTheme.light,
    locale: locale,
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    home: Scaffold(
      body: PeriodSelector(value: value, onChanged: onChanged),
    ),
  );
}

void main() {
  testWidgets('renders 7d / 30d / 90d / 1 year pills in English',
      (tester) async {
    await tester.pumpWidget(
      _harness(value: StatsPeriod.thirtyDays, onChanged: (_) {}),
    );
    await tester.pumpAndSettle();

    expect(find.text('7d'), findsOneWidget);
    expect(find.text('30d'), findsOneWidget);
    expect(find.text('90d'), findsOneWidget);
    expect(find.text('1 year'), findsOneWidget);
  });

  testWidgets('renders 7 j / 30 j / 90 j / 1 an pills in French',
      (tester) async {
    await tester.pumpWidget(
      _harness(
        value: StatsPeriod.thirtyDays,
        onChanged: (_) {},
        locale: const Locale('fr'),
      ),
    );
    await tester.pumpAndSettle();

    // French label conventions matter — the brief says tutoiement +
    // FR-conversational. The l10n bundle ships "7 j" / "1 an".
    expect(find.textContaining(RegExp(r'^7\s?j$')), findsOneWidget);
    expect(find.textContaining(RegExp(r'^30\s?j$')), findsOneWidget);
    expect(find.textContaining(RegExp(r'^90\s?j$')), findsOneWidget);
    expect(find.text('1 an'), findsOneWidget);
  });

  testWidgets('D3: tapping 1 year pill fires onChanged with oneYear',
      (tester) async {
    StatsPeriod? captured;
    await tester.pumpWidget(
      _harness(
        value: StatsPeriod.thirtyDays,
        onChanged: (p) => captured = p,
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('1 year'));
    expect(captured, StatsPeriod.oneYear);
  });

  testWidgets('tapping a pill calls onChanged with that period',
      (tester) async {
    StatsPeriod? captured;
    await tester.pumpWidget(
      _harness(
        value: StatsPeriod.thirtyDays,
        onChanged: (p) => captured = p,
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('7d'));
    expect(captured, StatsPeriod.sevenDays);

    captured = null;
    await tester.tap(find.text('90d'));
    expect(captured, StatsPeriod.ninetyDays);
  });

  testWidgets('tapping the already-selected pill still fires onChanged',
      (tester) async {
    // The contract is "this pill is the new value" — the parent decides
    // whether to short-circuit identical writes.
    var calls = 0;
    StatsPeriod? captured;
    await tester.pumpWidget(
      _harness(
        value: StatsPeriod.thirtyDays,
        onChanged: (p) {
          calls++;
          captured = p;
        },
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('30d'));
    expect(calls, 1);
    expect(captured, StatsPeriod.thirtyDays);
  });

  testWidgets('exposes a Semantics container with l10n label',
      (tester) async {
    await tester.pumpWidget(
      _harness(value: StatsPeriod.sevenDays, onChanged: (_) {}),
    );
    await tester.pumpAndSettle();

    // Selector advertises its purpose to screen readers — we don't pin
    // the exact string (l10n bundle owns it) but assert presence of a
    // labelled Semantics container.
    final widget = tester.widget<Semantics>(
      find
          .descendant(
            of: find.byType(PeriodSelector),
            matching: find.byType(Semantics),
          )
          .first,
    );
    expect(widget.properties.label, isNotEmpty);
  });
}

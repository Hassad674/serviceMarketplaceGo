import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/presentation/widgets/profile_identity_strip.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en'), Locale('fr')],
    locale: const Locale('en'),
    home: Scaffold(body: child),
  );
}

void main() {
  testWidgets('renders nothing when the profile has no Tier 1 data',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        ProfileIdentityStrip.fromProfileJson(const <String, dynamic>{}),
      ),
    );
    await tester.pumpAndSettle();

    // SizedBox.shrink() has no identifying text.
    expect(find.byType(Row), findsNothing);
  });

  testWidgets('renders availability + location blocks', (tester) async {
    await tester.pumpWidget(
      _wrap(
        ProfileIdentityStrip.fromProfileJson(<String, dynamic>{
          'availability_status': 'available_now',
          'city': 'Paris',
          'country_code': 'FR',
          'work_mode': ['remote'],
        }),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Available now'), findsOneWidget);
    expect(find.textContaining('Paris'), findsOneWidget);
    expect(find.textContaining('Remote'), findsOneWidget);
  });

  testWidgets('renders the pricing block for a direct daily row',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        ProfileIdentityStrip.fromProfileJson(<String, dynamic>{
          'pricing': [
            <String, dynamic>{
              'kind': 'direct',
              'type': 'daily',
              'min_amount': 50000,
              'max_amount': null,
              'currency': 'EUR',
              'note': '',
            },
          ],
        }),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('/day'), findsOneWidget);
  });

  testWidgets('shows two availability badges with referrer prefix',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        ProfileIdentityStrip.fromProfileJson(<String, dynamic>{
          'availability_status': 'available_now',
          'referrer_availability_status': 'available_soon',
        }),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Services'), findsOneWidget);
    expect(find.textContaining('Referrer'), findsOneWidget);
  });

  testWidgets('languages block shows flags with overflow indicator',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        ProfileIdentityStrip.fromProfileJson(<String, dynamic>{
          'languages_professional': ['fr', 'en', 'es', 'de', 'it', 'pt'],
        }),
      ),
    );
    await tester.pumpAndSettle();

    // 6 languages → 5 flags visible + "+1".
    expect(find.text('+1'), findsOneWidget);
  });
}

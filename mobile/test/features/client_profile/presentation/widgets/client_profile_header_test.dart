import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/client_profile/presentation/widgets/client_profile_header.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _host(Widget child) {
  return MaterialApp(
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('en')],
    home: Scaffold(body: SingleChildScrollView(child: child)),
  );
}

void main() {
  group('ClientProfileHeader', () {
    testWidgets('renders company name and stat labels', (tester) async {
      await tester.pumpWidget(_host(
        const ClientProfileHeader(
          companyName: 'Acme Corp',
          totalSpentCents: 123456,
          reviewCount: 4,
          averageRating: 4.5,
          projectsCompleted: 7,
        ),
      ));
      await tester.pumpAndSettle();

      expect(find.text('Acme Corp'), findsOneWidget);
      expect(find.text('Total spent'), findsOneWidget);
      expect(find.text('Reviews received'), findsOneWidget);
      expect(find.text('Average rating'), findsOneWidget);
      expect(find.text('Projects completed'), findsOneWidget);
      expect(find.text('7'), findsOneWidget);
    });

    testWidgets('formats large euro amounts with thousands separator',
        (tester) async {
      await tester.pumpWidget(_host(
        const ClientProfileHeader(
          companyName: 'Acme',
          totalSpentCents: 1234567,
          reviewCount: 0,
          averageRating: 0,
          projectsCompleted: 0,
        ),
      ));
      await tester.pumpAndSettle();

      // Prefix + grouped thousands — exact spacing character is
      // an implementation detail, so assert on the grouped form.
      expect(
        find.textContaining(RegExp(r'€12.?346')),
        findsOneWidget,
      );
    });

    testWidgets('shows em dash when rating is zero', (tester) async {
      await tester.pumpWidget(_host(
        const ClientProfileHeader(
          companyName: 'Acme',
          totalSpentCents: 0,
          reviewCount: 0,
          averageRating: 0,
          projectsCompleted: 0,
        ),
      ));
      await tester.pumpAndSettle();

      // The rating column renders — when there are no reviews.
      expect(find.text('—'), findsWidgets);
    });

    testWidgets(
      'renders the camera badge when onAvatarTap is provided',
      (tester) async {
        await tester.pumpWidget(_host(
          ClientProfileHeader(
            companyName: 'Acme',
            totalSpentCents: 0,
            reviewCount: 0,
            averageRating: 0,
            projectsCompleted: 0,
            onAvatarTap: () {},
          ),
        ));
        await tester.pumpAndSettle();

        expect(find.byIcon(Icons.camera_alt), findsOneWidget);
      },
    );

    testWidgets(
      'hides the camera badge when onAvatarTap is null',
      (tester) async {
        await tester.pumpWidget(_host(
          const ClientProfileHeader(
            companyName: 'Acme',
            totalSpentCents: 0,
            reviewCount: 0,
            averageRating: 0,
            projectsCompleted: 0,
          ),
        ));
        await tester.pumpAndSettle();

        expect(find.byIcon(Icons.camera_alt), findsNothing);
      },
    );

    testWidgets('renders the org-type badge when provided', (tester) async {
      await tester.pumpWidget(_host(
        const ClientProfileHeader(
          companyName: 'Acme',
          orgType: 'enterprise',
          totalSpentCents: 0,
          reviewCount: 0,
          averageRating: 0,
          projectsCompleted: 0,
        ),
      ));
      await tester.pumpAndSettle();

      expect(find.text('Enterprise'), findsOneWidget);
    });

    testWidgets(
      'falls back to "?" initials when company name is empty',
      (tester) async {
        await tester.pumpWidget(_host(
          const ClientProfileHeader(
            companyName: '',
            totalSpentCents: 0,
            reviewCount: 0,
            averageRating: 0,
            projectsCompleted: 0,
          ),
        ));
        await tester.pumpAndSettle();

        expect(find.text('?'), findsOneWidget);
      },
    );
  });
}

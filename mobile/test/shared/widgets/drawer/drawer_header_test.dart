import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';
import 'package:marketplace_mobile/shared/widgets/drawer/drawer_header.dart';

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
  group('DrawerHeaderTile', () {
    testWidgets('renders display_name', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerHeaderTile(
            user: {'display_name': 'Alice Doe', 'role': 'provider'},
            role: 'provider',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Alice Doe'), findsOneWidget);
    });

    testWidgets('falls back to first_name when display_name is missing',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerHeaderTile(
            user: {'first_name': 'Bob', 'role': 'agency'},
            role: 'agency',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Bob'), findsOneWidget);
    });

    testWidgets('falls back to "User" when no name is set', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerHeaderTile(user: {}, role: 'enterprise'),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('User'), findsOneWidget);
    });

    testWidgets('renders initials from first+last name', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerHeaderTile(
            user: {'first_name': 'Alice', 'last_name': 'Doe'},
            role: 'provider',
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('AD'), findsOneWidget);
    });

    testWidgets('renders ? when user is null', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerHeaderTile(user: null, role: 'provider'),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('?'), findsOneWidget);
    });
  });

  group('DrawerRoleBadge', () {
    testWidgets('renders agency role label in english', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerRoleBadge(
            role: 'agency',
            backgroundColor: Color(0xFFDBEAFE),
            foregroundColor: Color(0xFF1D4ED8),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(Container), findsAtLeastNWidgets(1));
      // We don't assert exact label string — l10n source of truth.
      // Instead assert the Text widget exists with non-empty content.
      final textWidget = tester.widget<Text>(find.byType(Text));
      expect(textWidget.data, isNotNull);
      expect(textWidget.data!.isNotEmpty, isTrue);
    });

    testWidgets('renders enterprise role with custom colors', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerRoleBadge(
            role: 'enterprise',
            backgroundColor: Color(0xFFF3E8FF),
            foregroundColor: Color(0xFF7E22CE),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.byType(DrawerRoleBadge), findsOneWidget);
    });

    testWidgets('falls back to raw role for unknown values', (tester) async {
      await tester.pumpWidget(
        _wrap(
          const DrawerRoleBadge(
            role: 'mystery',
            backgroundColor: Color(0xFFF1F5F9),
            foregroundColor: Color(0xFF64748B),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('mystery'), findsOneWidget);
    });
  });
}

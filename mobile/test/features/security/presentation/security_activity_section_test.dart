import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/core/theme/app_theme.dart';
import 'package:marketplace_mobile/features/security/domain/entities/security_activity_page.dart';
import 'package:marketplace_mobile/features/security/domain/entities/security_event.dart';
import 'package:marketplace_mobile/features/security/presentation/providers/security_providers.dart';
import 'package:marketplace_mobile/features/security/presentation/widgets/security_activity_section.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap({required SecurityActivityPage firstPage}) {
  return ProviderScope(
    overrides: [
      securityActivityProvider.overrideWith((ref, cursor) async {
        if (cursor == null) return firstPage;
        return const SecurityActivityPage(data: <SecurityEvent>[]);
      }),
    ],
    child: MaterialApp(
      theme: AppTheme.light,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('fr'),
      home: const Scaffold(body: SecurityActivitySection()),
    ),
  );
}

SecurityEvent _event({
  String id = 'evt-1',
  String action = 'auth.login_success',
  String userAgent = 'Ordinateur (Chrome 120)',
  SecurityAccessKind kind = SecurityAccessKind.desktop,
  String? ip = '203.0.113.4',
  DateTime? createdAt,
}) {
  return SecurityEvent(
    id: id,
    action: action,
    userAgentSummary: userAgent,
    accessKind: kind,
    createdAt: createdAt ?? DateTime.utc(2026, 5, 8, 12),
    ipAddress: ip,
  );
}

void main() {
  testWidgets('renders the empty state when the first page has no events',
      (tester) async {
    await tester.pumpWidget(
      _wrap(firstPage: const SecurityActivityPage(data: <SecurityEvent>[])),
    );
    await tester.pumpAndSettle();
    // FR copy from app_fr.arb.
    expect(
      find.text('Aucune activité récente à afficher.'),
      findsOneWidget,
    );
  });

  testWidgets('renders one row per event with device label and IP',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        firstPage: SecurityActivityPage(
          data: [
            _event(),
            _event(
              id: 'evt-2',
              action: 'auth.logout',
              userAgent: 'Mobile (Safari 16)',
              kind: SecurityAccessKind.mobile,
              ip: '198.51.100.7',
              createdAt: DateTime.utc(2026, 5, 8, 8),
            ),
          ],
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Ordinateur (Chrome 120)'), findsOneWidget);
    expect(find.text('Mobile (Safari 16)'), findsOneWidget);
    // The action + IP sit in the same Text node — match by substring.
    expect(
      find.textContaining('203.0.113.4'),
      findsWidgets,
    );
    expect(
      find.textContaining('198.51.100.7'),
      findsWidgets,
    );
  });

  testWidgets('falls back to the unknown-device label when summary is empty',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        firstPage: SecurityActivityPage(
          data: [_event(userAgent: '', kind: SecurityAccessKind.unknown)],
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Appareil inconnu'), findsOneWidget);
  });

  testWidgets('shows the "Voir plus" button when next_cursor is set',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        firstPage: SecurityActivityPage(
          data: [_event()],
          nextCursor: 'cur-2',
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Voir plus'), findsOneWidget);
  });
}

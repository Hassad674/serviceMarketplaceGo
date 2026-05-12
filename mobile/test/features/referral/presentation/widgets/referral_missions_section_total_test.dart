import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referral/domain/entities/referral_entity.dart';
import 'package:marketplace_mobile/features/referral/presentation/providers/referral_provider.dart';
import 'package:marketplace_mobile/features/referral/presentation/widgets/referral_missions_section.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child, {List<Override> overrides = const []}) {
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('fr'), Locale('en')],
      locale: const Locale('fr'),
      home: Scaffold(body: child),
    ),
  );
}

ReferralAttribution _attribution({
  required int totalAmountCents,
  String title = 'Mission alpha',
}) {
  return ReferralAttribution(
    id: 'att-1',
    proposalId: '00000000-0000-0000-0000-000000000001',
    proposalTitle: title,
    proposalStatus: 'active',
    totalAmountCents: totalAmountCents,
    ratePctSnapshot: 5,
    attributedAt: '2026-05-01T10:00:00Z',
    milestonesTotal: 2,
    milestonesPaid: 0,
    milestonesPending: 2,
  );
}

void main() {
  group('ReferralMissionsSection — mission total amount', () {
    testWidgets('renders the total amount pill when > 0', (tester) async {
      await tester.pumpWidget(_wrap(
        const ReferralMissionsSection(
          referralId: 'ref-1',
          viewerIsClient: false,
        ),
        overrides: [
          referralAttributionsProvider('ref-1').overrideWith((ref) async {
            return [_attribution(totalAmountCents: 123000)];
          }),
        ],
      ),);
      await tester.pumpAndSettle();
      // Pill is keyed for stable selection.
      expect(
        find.byKey(const ValueKey('attribution-total-amount')),
        findsOneWidget,
      );
      // Formatted amount visible.
      expect(find.text('1230 €'), findsOneWidget);
    });

    testWidgets('omits the pill when total amount is 0', (tester) async {
      await tester.pumpWidget(_wrap(
        const ReferralMissionsSection(
          referralId: 'ref-2',
          viewerIsClient: false,
        ),
        overrides: [
          referralAttributionsProvider('ref-2').overrideWith((ref) async {
            return [_attribution(totalAmountCents: 0)];
          }),
        ],
      ),);
      await tester.pumpAndSettle();
      expect(
        find.byKey(const ValueKey('attribution-total-amount')),
        findsNothing,
      );
    });

    testWidgets('also renders the pill for the client viewer (public price)',
        (tester) async {
      await tester.pumpWidget(_wrap(
        const ReferralMissionsSection(
          referralId: 'ref-3',
          viewerIsClient: true,
        ),
        overrides: [
          referralAttributionsProvider('ref-3').overrideWith((ref) async {
            return [_attribution(totalAmountCents: 50000)];
          }),
        ],
      ),);
      await tester.pumpAndSettle();
      // Client viewer sees the gross price but NOT the commission column —
      // the price is the public mission amount.
      expect(
        find.byKey(const ValueKey('attribution-total-amount')),
        findsOneWidget,
      );
      expect(find.text('500 €'), findsOneWidget);
    });
  });
}

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referral/domain/entities/referral_entity.dart';
import 'package:marketplace_mobile/features/referral/presentation/widgets/anonymized_client_card.dart';
import 'package:marketplace_mobile/features/referral/presentation/widgets/anonymized_provider_card.dart';
import 'package:marketplace_mobile/features/referral/presentation/widgets/referral_identity_card.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: const [Locale('fr'), Locale('en')],
    locale: const Locale('fr'),
    home: Scaffold(body: child),
  );
}

Referral _referral({
  String? providerDisplayName,
  String? clientDisplayName,
}) {
  return Referral(
    id: 'r1',
    referrerId: 'ref-1',
    providerId: 'prov-1',
    clientId: 'cli-1',
    durationMonths: 6,
    status: 'active',
    version: 1,
    introSnapshot: const IntroSnapshot(
      provider: ProviderSnapshot(
        expertiseDomains: ['SEO', 'Tech'],
      ),
      client: ClientSnapshot(industry: 'SaaS B2B'),
    ),
    lastActionAt: '2026-05-01T10:00:00Z',
    createdAt: '2026-05-01T10:00:00Z',
    updatedAt: '2026-05-01T10:00:00Z',
    providerDisplayName: providerDisplayName,
    clientDisplayName: clientDisplayName,
  );
}

void main() {
  group('ReferralIdentityCard — apporteur owner variant', () {
    testWidgets('renders ONLY the display names + role labels (minimalist)',
        (tester) async {
      await tester.pumpWidget(_wrap(ReferralIdentityCard(
        referral: _referral(),
        isOwner: true,
        providerName: 'Acme Consulting',
        clientName: 'Globex Corp',
      ),),);
      await tester.pumpAndSettle();
      // Clear provider + client names are visible.
      expect(find.text('Acme Consulting'), findsOneWidget);
      expect(find.text('Globex Corp'), findsOneWidget);
      // Role labels are visible.
      expect(find.text('Prestataire recommandé'), findsOneWidget);
      expect(find.text('Client proposé'), findsOneWidget);
      // Tile keys remain stable for future test selectors.
      expect(
        find.byKey(const ValueKey('referral-identity-provider')),
        findsOneWidget,
      );
      expect(
        find.byKey(const ValueKey('referral-identity-client')),
        findsOneWidget,
      );
      // Anonymized cards are NOT used in the owner branch.
      expect(find.byType(AnonymizedProviderCard), findsNothing);
      expect(find.byType(AnonymizedClientCard), findsNothing);
      // No legacy "Voir le profil" CTA or chevron — the minimalist
      // variant is purely informational.
      expect(find.text('Voir le profil du prestataire'), findsNothing);
      expect(find.text('Voir le profil du client'), findsNothing);
      expect(find.byIcon(Icons.arrow_forward_ios), findsNothing);
    });

    testWidgets('uses referral.providerDisplayName when no override is provided',
        (tester) async {
      await tester.pumpWidget(_wrap(ReferralIdentityCard(
        referral: _referral(
          providerDisplayName: 'Atelier Lumen',
          clientDisplayName: 'Banque du Sud',
        ),
        isOwner: true,
      ),),);
      await tester.pumpAndSettle();
      expect(find.text('Atelier Lumen'), findsOneWidget);
      expect(find.text('Banque du Sud'), findsOneWidget);
    });

    testWidgets('renders em-dash placeholder when no name is available',
        (tester) async {
      await tester.pumpWidget(_wrap(ReferralIdentityCard(
        referral: _referral(),
        isOwner: true,
      ),),);
      await tester.pumpAndSettle();
      // No name override + no display_name on referral → em-dash.
      // Should appear TWICE (provider tile + client tile).
      expect(find.text('—'), findsNWidgets(2));
      // The masked-card fallback to expertise/industry has been
      // removed — these strings MUST NOT leak into the revealed
      // variant.
      expect(find.text('SEO'), findsNothing);
      expect(find.text('SaaS B2B'), findsNothing);
    });
  });

  group('ReferralIdentityCard — masked variant', () {
    testWidgets('non-owner viewer → renders anonymized cards (regression)',
        (tester) async {
      await tester.pumpWidget(_wrap(ReferralIdentityCard(
        referral: _referral(),
        isOwner: false,
      ),),);
      await tester.pumpAndSettle();
      expect(find.byType(AnonymizedProviderCard), findsOneWidget);
      expect(find.byType(AnonymizedClientCard), findsOneWidget);
      expect(
        find.byKey(const ValueKey('referral-identity-provider')),
        findsNothing,
      );
    });
  });
}

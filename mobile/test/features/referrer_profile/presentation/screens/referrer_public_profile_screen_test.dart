import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_pricing.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_profile.dart';
import 'package:marketplace_mobile/features/referrer_profile/presentation/providers/referrer_profile_providers.dart';
import 'package:marketplace_mobile/features/referrer_profile/presentation/screens/referrer_public_profile_screen.dart';
import 'package:marketplace_mobile/features/referrer_reputation/domain/entities/referrer_reputation.dart';
import 'package:marketplace_mobile/features/referrer_reputation/presentation/providers/referrer_reputation_provider.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

const _emptyReputation = ReferrerReputation(
  ratingAvg: 0,
  reviewCount: 0,
  history: <ReferrerProjectHistoryEntry>[],
  nextCursor: '',
  hasMore: false,
);

Widget _buildTestable(
  ReferrerProfile profile, {
  String? displayName = 'Bob Connector',
  Locale locale = const Locale('en'),
  ReferrerReputation reputation = _emptyReputation,
}) {
  return ProviderScope(
    overrides: [
      referrerPublicProfileProvider('org-777').overrideWith(
        (ref) async => profile,
      ),
      // Override the reputation provider too so the public-profile
      // widget tree never reaches into the real Dio client during
      // tests — otherwise the ReferrerReputationWidget would render
      // its error state (no l10n strings present) and every assertion
      // on the empty-history copy would fail spuriously.
      referrerReputationProvider('org-777').overrideWith(
        (ref) async => reputation,
      ),
    ],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: locale,
      home: ReferrerPublicProfileScreen(
        organizationId: 'org-777',
        displayName: displayName,
      ),
    ),
  );
}

void main() {
  testWidgets('renders the referrer header, pricing and empty history',
      (tester) async {
    const profile = ReferrerProfile(
      id: 'r1',
      organizationId: 'org-777',
      title: 'Deal connector',
      about: 'Two decades of SaaS relationships.',
      videoUrl: '',
      availabilityStatus: 'available_now',
      expertiseDomains: <String>['sales'],
      photoUrl: '',
      city: 'Lyon',
      countryCode: 'FR',
      latitude: null,
      longitude: null,
      workMode: <String>[],
      travelRadiusKm: null,
      languagesProfessional: <String>['fr'],
      languagesConversational: <String>['en'],
      pricing: ReferrerPricing(
        type: ReferrerPricingType.commissionPct,
        minAmount: 800,
        maxAmount: null,
        currency: 'pct',
        note: '',
        negotiable: false,
      ),
    );

    await tester.pumpWidget(_buildTestable(profile));
    await tester.pumpAndSettle();

    expect(find.text('Bob Connector'), findsOneWidget);
    expect(find.text('Deal connector'), findsOneWidget);

    // Project history stays visible but renders the apporteur
    // reputation empty state — the public surface renders the
    // ReferrerReputationWidget, whose empty copy comes from
    // `reputationEmptyTitle` ("No referred project yet"), not the
    // legacy editable-screen copy.
    expect(find.textContaining('No referred project yet'), findsOneWidget);
  });

  testWidgets('never surfaces skills or portfolio on the referrer surface',
      (tester) async {
    const profile = ReferrerProfile(
      id: 'r1',
      organizationId: 'org-777',
      title: '',
      about: '',
      videoUrl: '',
      availabilityStatus: 'available_now',
      expertiseDomains: <String>[],
      photoUrl: '',
      city: '',
      countryCode: '',
      latitude: null,
      longitude: null,
      workMode: <String>[],
      travelRadiusKm: null,
      languagesProfessional: <String>[],
      languagesConversational: <String>[],
      pricing: null,
    );
    await tester.pumpWidget(_buildTestable(profile));
    await tester.pumpAndSettle();

    // No "Skills" section, no "Portfolio" section — the surface is
    // intentionally minimal on the referrer persona.
    expect(find.text('Skills'), findsNothing);
    expect(find.text('Portfolio'), findsNothing);
  });

  // Mirrors the web fix on /[locale]/referrers/[id]: when the
  // referrer has no display name AND no title, the header MUST fall
  // back to the localized "Business referrer" / "Apporteur d'affaires"
  // label. The raw organization id (the UUID in the URL) is never
  // surfaced. Production regression observed on
  // service-marketplace-go.vercel.app — same bug across surfaces.
  testWidgets('uses localized fallback name when title and explicit name are empty',
      (tester) async {
    const profile = ReferrerProfile(
      id: 'r1',
      organizationId: 'org-777',
      title: '',
      about: '',
      videoUrl: '',
      availabilityStatus: 'available_now',
      expertiseDomains: <String>[],
      photoUrl: '',
      city: '',
      countryCode: '',
      latitude: null,
      longitude: null,
      workMode: <String>[],
      travelRadiusKm: null,
      languagesProfessional: <String>[],
      languagesConversational: <String>[],
      pricing: null,
    );
    await tester.pumpWidget(_buildTestable(profile, displayName: ''));
    await tester.pumpAndSettle();

    expect(find.text('Business referrer'), findsWidgets);
    // Hard guarantee — the raw organization id must NEVER reach the
    // rendered DOM as the header text.
    expect(find.text('org-777'), findsNothing);
  });

  testWidgets('falls back to French label under fr locale',
      (tester) async {
    const profile = ReferrerProfile(
      id: 'r1',
      organizationId: 'org-777',
      title: '',
      about: '',
      videoUrl: '',
      availabilityStatus: 'available_now',
      expertiseDomains: <String>[],
      photoUrl: '',
      city: '',
      countryCode: '',
      latitude: null,
      longitude: null,
      workMode: <String>[],
      travelRadiusKm: null,
      languagesProfessional: <String>[],
      languagesConversational: <String>[],
      pricing: null,
    );
    await tester.pumpWidget(
      _buildTestable(
        profile,
        displayName: '',
        locale: const Locale('fr'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text("Apporteur d'affaires"), findsWidgets);
  });
}

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_pricing.dart';
import 'package:marketplace_mobile/features/referrer_profile/domain/entities/referrer_profile.dart';
import 'package:marketplace_mobile/features/referrer_profile/presentation/providers/referrer_profile_providers.dart';
import 'package:marketplace_mobile/features/referrer_profile/presentation/screens/referrer_public_profile_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _buildTestable(ReferrerProfile profile) {
  return ProviderScope(
    overrides: [
      referrerPublicProfileProvider('org-777').overrideWith(
        (ref) async => profile,
      ),
    ],
    child: const MaterialApp(
      localizationsDelegates: [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: [Locale('en'), Locale('fr')],
      locale: Locale('en'),
      home: ReferrerPublicProfileScreen(
        organizationId: 'org-777',
        displayName: 'Bob Connector',
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

    // Project history stays visible but renders the empty placeholder.
    expect(find.textContaining('No deals referred yet'), findsOneWidget);
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
}

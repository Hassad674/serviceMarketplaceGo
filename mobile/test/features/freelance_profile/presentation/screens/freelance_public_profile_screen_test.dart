import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_pricing.dart';
import 'package:marketplace_mobile/features/freelance_profile/domain/entities/freelance_profile.dart';
import 'package:marketplace_mobile/features/freelance_profile/presentation/providers/freelance_profile_providers.dart';
import 'package:marketplace_mobile/features/freelance_profile/presentation/screens/freelance_public_profile_screen.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _buildTestable(FreelanceProfile profile) {
  return ProviderScope(
    overrides: [
      freelancePublicProfileProvider('org-123').overrideWith(
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
      home: FreelancePublicProfileScreen(
        organizationId: 'org-123',
        displayName: 'Alice Doe',
      ),
    ),
  );
}

void main() {
  testWidgets('renders the freelance-specific sections including skills',
      (tester) async {
    const profile = FreelanceProfile(
      id: 'p1',
      organizationId: 'org-123',
      title: 'Full-stack engineer',
      about: 'I build marketplaces.',
      videoUrl: '',
      availabilityStatus: 'available_now',
      expertiseDomains: <String>['web_development'],
      photoUrl: '',
      city: 'Paris',
      countryCode: 'FR',
      latitude: null,
      longitude: null,
      workMode: <String>['remote'],
      travelRadiusKm: null,
      languagesProfessional: <String>['fr'],
      languagesConversational: <String>[],
      skills: <Map<String, dynamic>>[],
      pricing: FreelancePricing(
        type: FreelancePricingType.daily,
        minAmount: 50000,
        maxAmount: null,
        currency: 'EUR',
        note: '',
        negotiable: false,
      ),
    );

    await tester.pumpWidget(_buildTestable(profile));
    await tester.pumpAndSettle();

    expect(find.text('Alice Doe'), findsOneWidget);
    expect(find.text('Full-stack engineer'), findsOneWidget);
    expect(find.textContaining('Paris'), findsWidgets);
    expect(find.text('About'), findsOneWidget);
    expect(find.textContaining('I build marketplaces'), findsOneWidget);
  });

  testWidgets('shows empty pricing label when no pricing declared',
      (tester) async {
    const profile = FreelanceProfile(
      id: 'p1',
      organizationId: 'org-123',
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
      skills: <Map<String, dynamic>>[],
      pricing: null,
    );
    await tester.pumpWidget(_buildTestable(profile));
    await tester.pumpAndSettle();
    expect(find.textContaining('No pricing'), findsOneWidget);
  });
}

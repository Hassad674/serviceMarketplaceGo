import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:marketplace_mobile/features/payment_info/presentation/widgets/stripe_requirements_banner.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

/// Helper that wraps [StripeRequirementsBanner] in a [ProviderScope] with the
/// given [StripeRequirements] data, a [MaterialApp] for theming, and the
/// localization delegates so [AppLocalizations.of] resolves.
Widget _buildTestWidget(StripeRequirements requirements) {
  return ProviderScope(
    overrides: [
      stripeRequirementsProvider.overrideWith(
        (ref) => Future.value(requirements),
      ),
    ],
    child: const MaterialApp(
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      locale: Locale('en'),
      home: Scaffold(body: StripeRequirementsBanner()),
    ),
  );
}

void main() {
  group('StripeRequirementsBanner', () {
    testWidgets('renders nothing when no requirements', (tester) async {
      await tester.pumpWidget(
        _buildTestWidget(
          const StripeRequirements(
            hasRequirements: false,
            sections: [],
          ),
        ),
      );
      await tester.pumpAndSettle();

      // The widget should render SizedBox.shrink — no visible content.
      expect(find.byType(Container), findsNothing);
      expect(find.text('Additional information required'), findsNothing);
      expect(find.text('Eventually required'), findsNothing);
    });

    testWidgets('shows amber banner for eventually_due fields', (tester) async {
      await tester.pumpWidget(
        _buildTestWidget(
          StripeRequirements(
            hasRequirements: true,
            sections: [
              RequirementSection(
                id: 'personal',
                titleKey: 'personalInfo',
                fields: [
                  const RequirementField(
                    key: 'firstName',
                    labelKey: 'firstName',
                    urgency: 'eventually_due',
                  ),
                  const RequirementField(
                    key: 'lastName',
                    labelKey: 'lastName',
                    urgency: 'eventually_due',
                  ),
                ],
              ),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();

      // The amber (eventual) banner should be visible.
      expect(find.text('Eventually required'), findsOneWidget);

      // Amber background color (light mode): 0xFFFFFBEB.
      final amberContainer = tester.widgetList<Container>(
        find.byType(Container),
      ).where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && decoration.color != null) {
          return decoration.color == const Color(0xFFFFFBEB);
        }
        return false;
      });
      expect(amberContainer.isNotEmpty, isTrue);

      // No urgent (red) banner should appear.
      expect(find.text('Additional information required'), findsNothing);
    });

    testWidgets('shows red banner for currently_due fields', (tester) async {
      await tester.pumpWidget(
        _buildTestWidget(
          StripeRequirements(
            hasRequirements: true,
            sections: [
              RequirementSection(
                id: 'personal',
                titleKey: 'personalInfo',
                fields: [
                  const RequirementField(
                    key: 'email',
                    labelKey: 'email',
                    urgency: 'currently_due',
                  ),
                ],
              ),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();

      // The urgent (red) banner should be visible.
      expect(find.text('Additional information required'), findsOneWidget);

      // Red background color (light mode): 0xFFFEF2F2.
      final redContainer = tester.widgetList<Container>(
        find.byType(Container),
      ).where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && decoration.color != null) {
          return decoration.color == const Color(0xFFFEF2F2);
        }
        return false;
      });
      expect(redContainer.isNotEmpty, isTrue);
    });

    testWidgets('shows red banner for past_due fields', (tester) async {
      await tester.pumpWidget(
        _buildTestWidget(
          StripeRequirements(
            hasRequirements: true,
            sections: [
              RequirementSection(
                id: 'personal',
                titleKey: 'personalInfo',
                fields: [
                  const RequirementField(
                    key: 'address',
                    labelKey: 'address',
                    urgency: 'past_due',
                  ),
                ],
              ),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();

      // past_due maps to the urgent (red) banner.
      expect(find.text('Additional information required'), findsOneWidget);

      final redContainer = tester.widgetList<Container>(
        find.byType(Container),
      ).where((c) {
        final decoration = c.decoration;
        if (decoration is BoxDecoration && decoration.color != null) {
          return decoration.color == const Color(0xFFFEF2F2);
        }
        return false;
      });
      expect(redContainer.isNotEmpty, isTrue);
    });

    testWidgets('shows both banners when mixed urgencies', (tester) async {
      await tester.pumpWidget(
        _buildTestWidget(
          StripeRequirements(
            hasRequirements: true,
            sections: [
              RequirementSection(
                id: 'personal',
                titleKey: 'personalInfo',
                fields: [
                  const RequirementField(
                    key: 'firstName',
                    labelKey: 'firstName',
                    urgency: 'currently_due',
                  ),
                  const RequirementField(
                    key: 'lastName',
                    labelKey: 'lastName',
                    urgency: 'eventually_due',
                  ),
                ],
              ),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Both banners should be visible.
      expect(find.text('Additional information required'), findsOneWidget);
      expect(find.text('Eventually required'), findsOneWidget);

      // Verify both colored containers exist.
      final containers = tester.widgetList<Container>(
        find.byType(Container),
      );
      final hasRed = containers.any((c) {
        final decoration = c.decoration;
        return decoration is BoxDecoration &&
            decoration.color == const Color(0xFFFEF2F2);
      });
      final hasAmber = containers.any((c) {
        final decoration = c.decoration;
        return decoration is BoxDecoration &&
            decoration.color == const Color(0xFFFFFBEB);
      });
      expect(hasRed, isTrue);
      expect(hasAmber, isTrue);
    });

    testWidgets('displays deadline when present', (tester) async {
      // Unix timestamp 1735689600 = 2025-01-01 00:00:00 UTC
      await tester.pumpWidget(
        _buildTestWidget(
          StripeRequirements(
            hasRequirements: true,
            currentDeadline: 1735689600,
            sections: [
              RequirementSection(
                id: 'personal',
                titleKey: 'personalInfo',
                fields: [
                  const RequirementField(
                    key: 'email',
                    labelKey: 'email',
                    urgency: 'currently_due',
                  ),
                ],
              ),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();

      // The _formatDeadline method produces "d/m/yyyy" from a Unix timestamp.
      // 1735689600 -> 2025-01-01 UTC. In local time zone the day may vary,
      // so we just check that the "Deadline:" prefix and "2025" appear.
      expect(find.textContaining('Deadline:'), findsOneWidget);
      expect(find.textContaining('2025'), findsOneWidget);
    });

    testWidgets('displays field names in banner', (tester) async {
      await tester.pumpWidget(
        _buildTestWidget(
          StripeRequirements(
            hasRequirements: true,
            sections: [
              RequirementSection(
                id: 'personal',
                titleKey: 'personalInfo',
                fields: [
                  const RequirementField(
                    key: 'firstName',
                    labelKey: 'firstName',
                    urgency: 'currently_due',
                  ),
                  const RequirementField(
                    key: 'lastName',
                    labelKey: 'lastName',
                    urgency: 'currently_due',
                  ),
                  const RequirementField(
                    key: 'dateOfBirth',
                    labelKey: 'dateOfBirth',
                    urgency: 'eventually_due',
                  ),
                ],
              ),
            ],
          ),
        ),
      );
      await tester.pumpAndSettle();

      // _humanizeKey("firstName") -> "First Name"
      // _humanizeKey("lastName") -> "Last Name"
      // _humanizeKey("dateOfBirth") -> "Date Of Birth"
      // Fields are rendered with a bullet prefix: "\u2022 First Name"
      expect(find.textContaining('First Name'), findsOneWidget);
      expect(find.textContaining('Last Name'), findsOneWidget);
      expect(find.textContaining('Date Of Birth'), findsOneWidget);
    });
  });
}

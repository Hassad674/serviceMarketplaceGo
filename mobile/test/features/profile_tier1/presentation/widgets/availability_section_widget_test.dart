import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/profile_tier1/domain/entities/availability_status.dart';
import 'package:marketplace_mobile/features/profile_tier1/presentation/widgets/availability_section_widget.dart';
import 'package:marketplace_mobile/l10n/app_localizations.dart';

Widget _wrap(Widget child) {
  return ProviderScope(
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en'), Locale('fr')],
      locale: const Locale('en'),
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  testWidgets('renders a single badge when referrer is disabled',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        AvailabilitySectionWidget(
          initialDirect: AvailabilityStatus.availableNow,
          initialReferrer: null,
          referrerEnabled: false,
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Available now'), findsOneWidget);
    // When referrer is disabled, the referral prefix must not render.
    expect(find.textContaining('Referrer'), findsNothing);
  });

  testWidgets('renders two prefixed badges when referrer is enabled',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        AvailabilitySectionWidget(
          initialDirect: AvailabilityStatus.availableSoon,
          initialReferrer: AvailabilityStatus.notAvailable,
          referrerEnabled: true,
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Services'), findsOneWidget);
    expect(find.textContaining('Referrer'), findsOneWidget);
    expect(find.textContaining('Available soon'), findsOneWidget);
    expect(find.textContaining('Unavailable'), findsOneWidget);
  });

  testWidgets('hides edit button when canEdit is false', (tester) async {
    await tester.pumpWidget(
      _wrap(
        AvailabilitySectionWidget(
          initialDirect: AvailabilityStatus.availableNow,
          initialReferrer: null,
          referrerEnabled: false,
          canEdit: false,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Update availability'), findsNothing);
  });

  testWidgets('AvailabilityBadge renders a colored dot + label',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        const AvailabilityBadge(
          status: AvailabilityStatus.availableNow,
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Available now'), findsOneWidget);
  });
}

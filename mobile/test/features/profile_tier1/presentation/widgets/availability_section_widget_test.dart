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
  testWidgets('direct variant renders only the direct badge',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        AvailabilitySectionWidget(
          variant: AvailabilityVariant.direct,
          initialDirect: AvailabilityStatus.availableNow,
          initialReferrer: AvailabilityStatus.notAvailable,
          referrerEnabled: true,
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Available now'), findsOneWidget);
    expect(find.text('Unavailable'), findsNothing);
  });

  testWidgets('referrer variant renders only the referrer badge',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        AvailabilitySectionWidget(
          variant: AvailabilityVariant.referrer,
          initialDirect: AvailabilityStatus.availableNow,
          initialReferrer: AvailabilityStatus.notAvailable,
          referrerEnabled: true,
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Unavailable'), findsOneWidget);
    expect(find.text('Available now'), findsNothing);
  });

  testWidgets('referrer variant self-hides when referrer disabled',
      (tester) async {
    await tester.pumpWidget(
      _wrap(
        AvailabilitySectionWidget(
          variant: AvailabilityVariant.referrer,
          initialDirect: AvailabilityStatus.availableNow,
          initialReferrer: null,
          referrerEnabled: false,
          canEdit: true,
          onSaved: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Availability'), findsNothing);
  });

  testWidgets('hides edit button when canEdit is false', (tester) async {
    await tester.pumpWidget(
      _wrap(
        AvailabilitySectionWidget(
          variant: AvailabilityVariant.direct,
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

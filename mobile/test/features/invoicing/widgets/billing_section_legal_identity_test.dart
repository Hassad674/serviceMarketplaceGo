import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/invoicing/domain/entities/billing_profile.dart';
import 'package:marketplace_mobile/features/invoicing/presentation/widgets/billing_section_legal_identity.dart';

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('BillingProfileTypeRadio', () {
    testWidgets('individual selection renders one checked + one unchecked',
        (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingProfileTypeRadio(
            value: ProfileType.individual,
            onChanged: (_) {},
          ),
        ),
      );
      expect(find.text('Particulier'), findsOneWidget);
      expect(find.text('Entreprise'), findsOneWidget);
      expect(find.byIcon(Icons.radio_button_checked), findsOneWidget);
      expect(find.byIcon(Icons.radio_button_unchecked), findsOneWidget);
    });

    testWidgets('null value renders both unchecked', (tester) async {
      await tester.pumpWidget(
        _wrap(
          BillingProfileTypeRadio(value: null, onChanged: (_) {}),
        ),
      );
      expect(find.byIcon(Icons.radio_button_checked), findsNothing);
      expect(find.byIcon(Icons.radio_button_unchecked), findsNWidgets(2));
    });

    testWidgets('tap "Entreprise" calls onChanged with business',
        (tester) async {
      ProfileType? captured;
      await tester.pumpWidget(
        _wrap(
          BillingProfileTypeRadio(
            value: null,
            onChanged: (v) => captured = v,
          ),
        ),
      );
      await tester.tap(find.text('Entreprise'));
      await tester.pump();
      expect(captured, ProfileType.business);
    });
  });

  group('BillingLegalIdentitySection', () {
    testWidgets('isBusiness=false hides trading + legal form fields',
        (tester) async {
      final ln = TextEditingController();
      final tn = TextEditingController();
      final lf = TextEditingController();
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingLegalIdentitySection(
              isBusiness: false,
              legalName: ln,
              tradingName: tn,
              legalForm: lf,
            ),
          ),
        ),
      );
      expect(find.text('Identité légale'), findsOneWidget);
      expect(find.text('Raison sociale ou nom légal'), findsOneWidget);
      expect(find.text('Nom commercial (optionnel)'), findsNothing);
      expect(find.text('Forme juridique'), findsNothing);
    });

    testWidgets('isBusiness=true shows trading + legal form fields',
        (tester) async {
      final ln = TextEditingController();
      final tn = TextEditingController();
      final lf = TextEditingController();
      await tester.pumpWidget(
        _wrap(
          Form(
            child: BillingLegalIdentitySection(
              isBusiness: true,
              legalName: ln,
              tradingName: tn,
              legalForm: lf,
            ),
          ),
        ),
      );
      expect(find.text('Nom commercial (optionnel)'), findsOneWidget);
      expect(find.text('Forme juridique'), findsOneWidget);
      expect(find.text('SAS, SARL, EURL, etc.'), findsOneWidget);
    });
  });
}

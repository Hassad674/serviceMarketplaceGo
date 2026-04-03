import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:marketplace_mobile/features/payment_info/lib/form_data_mapper.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/payment_info_entity.dart';
import 'helpers/kyc_test_infra.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('debug: responseToFormData values', (tester) async {
    final entity = PaymentInfo(
      id: 'test',
      userId: 'user1',
      firstName: 'Pierre',
      lastName: 'Martin',
      dateOfBirth: '1990-05-15',
      nationality: 'FR',
      address: '42 Rue Lafayette',
      city: 'Paris',
      postalCode: '75009',
      phone: '+33600112233',
      accountHolder: 'Pierre Martin',
      country: 'FR',
      iban: testIban,
      bic: testBic,
      bankCountry: 'FR',
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
    );

    final formData = responseToFormData(entity);
    debugPrint('country: ${formData.country}');
    debugPrint('values entries: ${formData.values.length}');
    for (final e in formData.values.entries) {
      debugPrint('  ${e.key} = ${e.value}');
    }

    // Check if the country fields mock returns sections
    final repo = InMemoryPaymentInfoRepository();
    final fields = await repo.getCountryFields('FR', 'individual');
    debugPrint('sections: ${fields.sections.length}');
    for (final s in fields.sections) {
      debugPrint('  section: ${s.id} (${s.titleKey}) fields: ${s.fields.length}');
      for (final f in s.fields) {
        final val = formData.values[f.key] ?? '<MISSING>';
        debugPrint('    ${f.key} -> $val');
      }
    }

    expect(formData.values['individual.first_name'], 'Pierre');
    expect(formData.country, 'FR');
  });
}

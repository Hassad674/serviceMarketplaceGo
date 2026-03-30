import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/payment_info/types/payment_info.dart';

void main() {
  group('PaymentInfoFormData', () {
    test('default values match expected KYC-safe defaults', () {
      const data = PaymentInfoFormData();

      expect(data.isBusiness, false);
      expect(data.firstName, '');
      expect(data.lastName, '');
      expect(data.dateOfBirth, '');
      expect(data.nationality, '');
      expect(data.address, '');
      expect(data.city, '');
      expect(data.postalCode, '');
      expect(data.phone, '');
      expect(data.activitySector, '8999');
      expect(data.businessRole, isNull);
      expect(data.businessName, '');
      expect(data.businessAddress, '');
      expect(data.businessCity, '');
      expect(data.businessPostalCode, '');
      expect(data.businessCountry, '');
      expect(data.taxId, '');
      expect(data.vatNumber, '');
      expect(data.isSelfRepresentative, true);
      expect(data.isSelfDirector, true);
      expect(data.noMajorOwners, true);
      expect(data.isSelfExecutive, true);
      expect(data.businessPersons, isEmpty);
      expect(data.bankMode, BankAccountMode.iban);
      expect(data.iban, '');
      expect(data.bic, '');
      expect(data.accountNumber, '');
      expect(data.routingNumber, '');
      expect(data.accountHolder, '');
      expect(data.bankCountry, '');
    });

    test('copyWith preserves unchanged values', () {
      const original = PaymentInfoFormData(
        firstName: 'Jean',
        lastName: 'Dupont',
        dateOfBirth: '1990-01-01',
        nationality: 'FR',
        address: '10 Rue',
        city: 'Paris',
        postalCode: '75001',
        phone: '+33612345678',
        activitySector: '7372',
        iban: 'FR76123456',
        accountHolder: 'Jean Dupont',
        bankCountry: 'FR',
      );

      final updated = original.copyWith(firstName: 'Pierre');

      expect(updated.firstName, 'Pierre');
      // All other fields remain unchanged
      expect(updated.lastName, 'Dupont');
      expect(updated.dateOfBirth, '1990-01-01');
      expect(updated.nationality, 'FR');
      expect(updated.address, '10 Rue');
      expect(updated.city, 'Paris');
      expect(updated.postalCode, '75001');
      expect(updated.phone, '+33612345678');
      expect(updated.activitySector, '7372');
      expect(updated.iban, 'FR76123456');
      expect(updated.accountHolder, 'Jean Dupont');
      expect(updated.bankCountry, 'FR');
    });

    test('copyWith updates multiple fields at once', () {
      const original = PaymentInfoFormData();

      final updated = original.copyWith(
        isBusiness: true,
        businessName: 'Test Corp',
        businessRole: BusinessRole.ceo,
        taxId: '123456',
      );

      expect(updated.isBusiness, true);
      expect(updated.businessName, 'Test Corp');
      expect(updated.businessRole, BusinessRole.ceo);
      expect(updated.taxId, '123456');
    });

    test('copyWith replaces business persons list', () {
      const original = PaymentInfoFormData();
      const person = BusinessPerson(
        role: 'director',
        firstName: 'Alice',
        lastName: 'Noir',
      );

      final updated = original.copyWith(businessPersons: [person]);

      expect(updated.businessPersons, hasLength(1));
      expect(updated.businessPersons.first.firstName, 'Alice');
    });

    test('copyWith switches bank mode', () {
      const original = PaymentInfoFormData(
        bankMode: BankAccountMode.iban,
        iban: 'FR761234',
      );

      final updated = original.copyWith(
        bankMode: BankAccountMode.local,
        accountNumber: '1234567890',
        routingNumber: '021000089',
      );

      expect(updated.bankMode, BankAccountMode.local);
      expect(updated.accountNumber, '1234567890');
      expect(updated.routingNumber, '021000089');
      // Old IBAN remains (not cleared by copyWith — this is by design)
      expect(updated.iban, 'FR761234');
    });
  });

  group('BusinessPerson', () {
    test('toJson includes all fields with snake_case keys', () {
      const person = BusinessPerson(
        role: 'representative',
        firstName: 'Marie',
        lastName: 'Durand',
        dateOfBirth: '1988-03-15',
        email: 'marie@test.com',
        phone: '+33611223344',
        address: '5 Boulevard',
        city: 'Lyon',
        postalCode: '69001',
        title: 'CEO',
      );

      final json = person.toJson();

      expect(json['role'], 'representative');
      expect(json['first_name'], 'Marie');
      expect(json['last_name'], 'Durand');
      expect(json['date_of_birth'], '1988-03-15');
      expect(json['email'], 'marie@test.com');
      expect(json['phone'], '+33611223344');
      expect(json['address'], '5 Boulevard');
      expect(json['city'], 'Lyon');
      expect(json['postal_code'], '69001');
      expect(json['title'], 'CEO');
    });

    test('toJson includes empty strings for default fields', () {
      const person = BusinessPerson();

      final json = person.toJson();

      expect(json['role'], 'director');
      expect(json['first_name'], '');
      expect(json['last_name'], '');
      expect(json['date_of_birth'], '');
      expect(json['email'], '');
      expect(json['phone'], '');
      expect(json['address'], '');
      expect(json['city'], '');
      expect(json['postal_code'], '');
      expect(json['title'], '');
    });

    test('default values match expected defaults', () {
      const person = BusinessPerson();

      expect(person.role, 'director');
      expect(person.firstName, '');
      expect(person.lastName, '');
      expect(person.dateOfBirth, '');
      expect(person.email, '');
      expect(person.phone, '');
      expect(person.address, '');
      expect(person.city, '');
      expect(person.postalCode, '');
      expect(person.title, '');
    });

    test('copyWith preserves unchanged fields', () {
      const original = BusinessPerson(
        role: 'owner',
        firstName: 'Jean',
        lastName: 'Martin',
        email: 'jean@test.com',
      );

      final updated = original.copyWith(firstName: 'Pierre');

      expect(updated.firstName, 'Pierre');
      expect(updated.role, 'owner');
      expect(updated.lastName, 'Martin');
      expect(updated.email, 'jean@test.com');
    });

    test('copyWith updates role', () {
      const original = BusinessPerson(role: 'director');

      final updated = original.copyWith(role: 'executive');

      expect(updated.role, 'executive');
    });
  });

  group('BankAccountMode', () {
    test('has iban and local values', () {
      expect(BankAccountMode.values, hasLength(2));
      expect(BankAccountMode.values, contains(BankAccountMode.iban));
      expect(BankAccountMode.values, contains(BankAccountMode.local));
    });
  });

  group('BusinessRole', () {
    test('has all five expected values', () {
      expect(BusinessRole.values, hasLength(5));
      expect(BusinessRole.values, contains(BusinessRole.owner));
      expect(BusinessRole.values, contains(BusinessRole.ceo));
      expect(BusinessRole.values, contains(BusinessRole.director));
      expect(BusinessRole.values, contains(BusinessRole.partner));
      expect(BusinessRole.values, contains(BusinessRole.other));
    });
  });
}

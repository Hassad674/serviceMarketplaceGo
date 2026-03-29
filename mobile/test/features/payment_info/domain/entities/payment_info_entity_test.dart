import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/payment_info_entity.dart';

void main() {
  group('PaymentInfo', () {
    test('creates with required fields and correct defaults', () {
      final info = PaymentInfo(
        id: 'pi-1',
        userId: 'user-1',
        firstName: 'John',
        lastName: 'Doe',
        dateOfBirth: '1990-01-15',
        nationality: 'FR',
        address: '1 Rue Test',
        city: 'Paris',
        postalCode: '75001',
        accountHolder: 'John Doe',
        createdAt: DateTime.utc(2026, 3, 27),
        updatedAt: DateTime.utc(2026, 3, 27),
      );

      expect(info.id, 'pi-1');
      expect(info.firstName, 'John');
      expect(info.lastName, 'Doe');
      expect(info.isBusiness, false);
      expect(info.businessName, '');
      expect(info.businessAddress, '');
      expect(info.businessCity, '');
      expect(info.businessPostalCode, '');
      expect(info.businessCountry, '');
      expect(info.taxId, '');
      expect(info.vatNumber, '');
      expect(info.roleInCompany, '');
      expect(info.iban, '');
      expect(info.bic, '');
      expect(info.accountNumber, '');
      expect(info.routingNumber, '');
      expect(info.bankCountry, '');
      expect(info.stripeAccountId, '');
      expect(info.stripeVerified, false);
    });

    test('creates with business fields', () {
      final info = PaymentInfo(
        id: 'pi-2',
        userId: 'user-2',
        firstName: 'Jane',
        lastName: 'Smith',
        dateOfBirth: '1985-06-20',
        nationality: 'DE',
        address: '10 Berliner Str',
        city: 'Berlin',
        postalCode: '10115',
        isBusiness: true,
        businessName: 'Smith GmbH',
        businessAddress: '10 Berliner Str',
        businessCity: 'Berlin',
        businessPostalCode: '10115',
        businessCountry: 'DE',
        taxId: 'DE123456789',
        vatNumber: 'DE987654321',
        roleInCompany: 'director',
        accountHolder: 'Smith GmbH',
        createdAt: DateTime.utc(2026, 3, 27),
        updatedAt: DateTime.utc(2026, 3, 27),
      );

      expect(info.isBusiness, true);
      expect(info.businessName, 'Smith GmbH');
      expect(info.roleInCompany, 'director');
      expect(info.taxId, 'DE123456789');
    });

    test('fromJson parses all fields correctly', () {
      final json = {
        'id': 'pi-10',
        'user_id': 'user-5',
        'first_name': 'Alice',
        'last_name': 'Martin',
        'date_of_birth': '1992-08-10',
        'nationality': 'FR',
        'address': '5 Avenue Foch',
        'city': 'Lyon',
        'postal_code': '69001',
        'is_business': true,
        'business_name': 'Martin SAS',
        'business_address': '5 Avenue Foch',
        'business_city': 'Lyon',
        'business_postal_code': '69001',
        'business_country': 'FR',
        'tax_id': 'FR12345678901',
        'vat_number': 'FR98765432101',
        'role_in_company': 'representative',
        'iban': 'FR7630006000011234567890189',
        'bic': 'BNPAFRPP',
        'account_number': '1234567890',
        'routing_number': '110000000',
        'account_holder': 'Martin SAS',
        'bank_country': 'FR',
        'stripe_account_id': 'acct_1234',
        'stripe_verified': true,
        'created_at': '2026-03-27T10:00:00Z',
        'updated_at': '2026-03-28T08:00:00Z',
      };

      final info = PaymentInfo.fromJson(json);

      expect(info.id, 'pi-10');
      expect(info.userId, 'user-5');
      expect(info.firstName, 'Alice');
      expect(info.lastName, 'Martin');
      expect(info.dateOfBirth, '1992-08-10');
      expect(info.nationality, 'FR');
      expect(info.address, '5 Avenue Foch');
      expect(info.city, 'Lyon');
      expect(info.postalCode, '69001');
      expect(info.isBusiness, true);
      expect(info.businessName, 'Martin SAS');
      expect(info.businessAddress, '5 Avenue Foch');
      expect(info.businessCity, 'Lyon');
      expect(info.businessPostalCode, '69001');
      expect(info.businessCountry, 'FR');
      expect(info.taxId, 'FR12345678901');
      expect(info.vatNumber, 'FR98765432101');
      expect(info.roleInCompany, 'representative');
      expect(info.iban, 'FR7630006000011234567890189');
      expect(info.bic, 'BNPAFRPP');
      expect(info.accountNumber, '1234567890');
      expect(info.routingNumber, '110000000');
      expect(info.accountHolder, 'Martin SAS');
      expect(info.bankCountry, 'FR');
      expect(info.stripeAccountId, 'acct_1234');
      expect(info.stripeVerified, true);
      expect(info.createdAt, DateTime.utc(2026, 3, 27, 10));
      expect(info.updatedAt, DateTime.utc(2026, 3, 28, 8));
    });

    test('fromJson handles missing optional fields', () {
      final json = {
        'id': 'pi-11',
        'user_id': 'user-5',
        'first_name': 'Bob',
        'last_name': 'Jones',
        'date_of_birth': '1988-02-14',
        'nationality': 'US',
        'address': '100 Main St',
        'city': 'NYC',
        'postal_code': '10001',
        'account_holder': 'Bob Jones',
        'created_at': '2026-03-27T10:00:00Z',
        'updated_at': '2026-03-27T10:00:00Z',
      };

      final info = PaymentInfo.fromJson(json);

      expect(info.isBusiness, false);
      expect(info.businessName, '');
      expect(info.businessAddress, '');
      expect(info.businessCity, '');
      expect(info.businessPostalCode, '');
      expect(info.businessCountry, '');
      expect(info.taxId, '');
      expect(info.vatNumber, '');
      expect(info.roleInCompany, '');
      expect(info.iban, '');
      expect(info.bic, '');
      expect(info.accountNumber, '');
      expect(info.routingNumber, '');
      expect(info.bankCountry, '');
      expect(info.stripeAccountId, '');
      expect(info.stripeVerified, false);
    });

    test('fromJson handles null optional fields', () {
      final json = {
        'id': 'pi-12',
        'user_id': 'user-5',
        'first_name': 'Carol',
        'last_name': 'White',
        'date_of_birth': '1995-11-30',
        'nationality': 'GB',
        'address': '10 Baker St',
        'city': 'London',
        'postal_code': 'NW1 6XE',
        'is_business': null,
        'business_name': null,
        'iban': null,
        'stripe_account_id': null,
        'stripe_verified': null,
        'account_holder': 'Carol White',
        'created_at': '2026-03-27T10:00:00Z',
        'updated_at': '2026-03-27T10:00:00Z',
      };

      final info = PaymentInfo.fromJson(json);

      expect(info.isBusiness, false);
      expect(info.businessName, '');
      expect(info.iban, '');
      expect(info.stripeAccountId, '');
      expect(info.stripeVerified, false);
    });
  });

  group('PaymentInfoStatus', () {
    test('creates with complete true', () {
      const status = PaymentInfoStatus(complete: true);

      expect(status.complete, true);
    });

    test('creates with complete false', () {
      const status = PaymentInfoStatus(complete: false);

      expect(status.complete, false);
    });

    test('fromJson parses complete true', () {
      final json = {'complete': true};

      final status = PaymentInfoStatus.fromJson(json);

      expect(status.complete, true);
    });

    test('fromJson parses complete false', () {
      final json = {'complete': false};

      final status = PaymentInfoStatus.fromJson(json);

      expect(status.complete, false);
    });
  });
}

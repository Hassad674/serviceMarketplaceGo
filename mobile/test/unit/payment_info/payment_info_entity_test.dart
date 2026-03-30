import 'package:flutter_test/flutter_test.dart';
import 'package:marketplace_mobile/features/payment_info/domain/entities/payment_info_entity.dart';

void main() {
  group('PaymentInfo.fromJson', () {
    test('parses complete JSON with all fields', () {
      final json = <String, dynamic>{
        'id': 'pi-123',
        'user_id': 'user-456',
        'first_name': 'Jean',
        'last_name': 'Dupont',
        'date_of_birth': '1990-05-15',
        'nationality': 'FR',
        'address': '10 Rue de Rivoli',
        'city': 'Paris',
        'postal_code': '75001',
        'phone': '+33612345678',
        'activity_sector': '7372',
        'is_business': true,
        'business_name': 'Dupont SARL',
        'business_address': '20 Avenue des Champs',
        'business_city': 'Paris',
        'business_postal_code': '75008',
        'business_country': 'FR',
        'tax_id': '123456789',
        'vat_number': 'FR12345678901',
        'role_in_company': 'ceo',
        'is_self_representative': false,
        'is_self_director': false,
        'no_major_owners': false,
        'is_self_executive': false,
        'iban': 'FR7612345678901234567890123',
        'bic': 'BNPAFRPP',
        'account_number': '',
        'routing_number': '',
        'account_holder': 'Jean Dupont',
        'bank_country': 'FR',
        'stripe_account_id': 'acct_abc123',
        'stripe_verified': true,
        'created_at': '2026-01-01T00:00:00Z',
        'updated_at': '2026-03-15T12:30:00Z',
      };

      final info = PaymentInfo.fromJson(json);

      expect(info.id, 'pi-123');
      expect(info.userId, 'user-456');
      expect(info.firstName, 'Jean');
      expect(info.lastName, 'Dupont');
      expect(info.dateOfBirth, '1990-05-15');
      expect(info.nationality, 'FR');
      expect(info.address, '10 Rue de Rivoli');
      expect(info.city, 'Paris');
      expect(info.postalCode, '75001');
      expect(info.phone, '+33612345678');
      expect(info.activitySector, '7372');
      expect(info.isBusiness, true);
      expect(info.businessName, 'Dupont SARL');
      expect(info.businessAddress, '20 Avenue des Champs');
      expect(info.businessCity, 'Paris');
      expect(info.businessPostalCode, '75008');
      expect(info.businessCountry, 'FR');
      expect(info.taxId, '123456789');
      expect(info.vatNumber, 'FR12345678901');
      expect(info.roleInCompany, 'ceo');
      expect(info.isSelfRepresentative, false);
      expect(info.isSelfDirector, false);
      expect(info.noMajorOwners, false);
      expect(info.isSelfExecutive, false);
      expect(info.iban, 'FR7612345678901234567890123');
      expect(info.bic, 'BNPAFRPP');
      expect(info.accountNumber, '');
      expect(info.routingNumber, '');
      expect(info.accountHolder, 'Jean Dupont');
      expect(info.bankCountry, 'FR');
      expect(info.stripeAccountId, 'acct_abc123');
      expect(info.stripeVerified, true);
      expect(info.createdAt, DateTime.parse('2026-01-01T00:00:00Z'));
      expect(info.updatedAt, DateTime.parse('2026-03-15T12:30:00Z'));
    });

    test('uses defaults for missing optional fields', () {
      final json = <String, dynamic>{
        'id': 'pi-minimal',
        'user_id': 'user-789',
        'first_name': 'Marie',
        'last_name': 'Martin',
        'date_of_birth': '1985-12-01',
        'nationality': 'DE',
        'address': 'Hauptstrasse 1',
        'city': 'Berlin',
        'postal_code': '10115',
        'account_holder': 'Marie Martin',
        'created_at': '2026-02-01T00:00:00Z',
        'updated_at': '2026-02-01T00:00:00Z',
      };

      final info = PaymentInfo.fromJson(json);

      // Optional fields should default
      expect(info.phone, '');
      expect(info.activitySector, '8999');
      expect(info.isBusiness, false);
      expect(info.businessName, '');
      expect(info.businessAddress, '');
      expect(info.businessCity, '');
      expect(info.businessPostalCode, '');
      expect(info.businessCountry, '');
      expect(info.taxId, '');
      expect(info.vatNumber, '');
      expect(info.roleInCompany, '');
      expect(info.isSelfRepresentative, true);
      expect(info.isSelfDirector, true);
      expect(info.noMajorOwners, true);
      expect(info.isSelfExecutive, true);
      expect(info.iban, '');
      expect(info.bic, '');
      expect(info.accountNumber, '');
      expect(info.routingNumber, '');
      expect(info.bankCountry, '');
      expect(info.stripeAccountId, '');
      expect(info.stripeVerified, false);
    });

    test('uses defaults when optional fields are explicitly null', () {
      final json = <String, dynamic>{
        'id': 'pi-null-test',
        'user_id': 'user-null',
        'first_name': 'Alice',
        'last_name': 'Noir',
        'date_of_birth': '1992-06-20',
        'nationality': 'FR',
        'address': '5 Rue Test',
        'city': 'Lyon',
        'postal_code': '69001',
        'phone': null,
        'activity_sector': null,
        'is_business': null,
        'business_name': null,
        'is_self_representative': null,
        'is_self_director': null,
        'no_major_owners': null,
        'is_self_executive': null,
        'iban': null,
        'bic': null,
        'stripe_account_id': null,
        'stripe_verified': null,
        'account_holder': 'Alice Noir',
        'created_at': '2026-03-01T00:00:00Z',
        'updated_at': '2026-03-01T00:00:00Z',
      };

      final info = PaymentInfo.fromJson(json);

      expect(info.phone, '');
      expect(info.activitySector, '8999');
      expect(info.isBusiness, false);
      expect(info.businessName, '');
      expect(info.isSelfRepresentative, true);
      expect(info.isSelfDirector, true);
      expect(info.noMajorOwners, true);
      expect(info.isSelfExecutive, true);
      expect(info.iban, '');
      expect(info.bic, '');
      expect(info.stripeAccountId, '');
      expect(info.stripeVerified, false);
    });
  });

  group('PaymentInfoStatus.fromJson', () {
    test('parses complete status as true', () {
      final status = PaymentInfoStatus.fromJson({'complete': true});
      expect(status.complete, true);
    });

    test('parses incomplete status as false', () {
      final status = PaymentInfoStatus.fromJson({'complete': false});
      expect(status.complete, false);
    });
  });
}

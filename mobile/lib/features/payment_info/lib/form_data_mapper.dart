import '../domain/entities/country_field_spec.dart';
import '../domain/entities/payment_info_entity.dart';
import '../types/payment_info.dart';

/// Convert a flat API [PaymentInfo] entity into a [PaymentInfoFormData]
/// with path-keyed [values] for dynamic section rendering.
///
/// Mirrors the web's `responseToFormData` function.
PaymentInfoFormData responseToFormData(PaymentInfo info) {
  final isBusiness = info.isBusiness;
  final prefix = isBusiness ? 'representative' : 'individual';
  final hasIban = info.iban.isNotEmpty;

  final values = <String, String>{};

  // Personal / representative fields
  if (info.firstName.isNotEmpty) values['$prefix.first_name'] = info.firstName;
  if (info.lastName.isNotEmpty) values['$prefix.last_name'] = info.lastName;
  if (info.dateOfBirth.isNotEmpty) values['$prefix.dob'] = info.dateOfBirth;
  if (info.nationality.isNotEmpty) {
    values['$prefix.nationality'] = info.nationality;
  }
  if (info.address.isNotEmpty) {
    values['$prefix.address.line1'] = info.address;
  }
  if (info.city.isNotEmpty) values['$prefix.address.city'] = info.city;
  if (info.postalCode.isNotEmpty) {
    values['$prefix.address.postal_code'] = info.postalCode;
  }
  if (info.phone.isNotEmpty) values['$prefix.phone'] = info.phone;

  // Company fields
  if (isBusiness) {
    if (info.businessName.isNotEmpty) values['company.name'] = info.businessName;
    if (info.businessAddress.isNotEmpty) {
      values['company.address.line1'] = info.businessAddress;
    }
    if (info.businessCity.isNotEmpty) {
      values['company.address.city'] = info.businessCity;
    }
    if (info.businessPostalCode.isNotEmpty) {
      values['company.address.postal_code'] = info.businessPostalCode;
    }
    if (info.businessCountry.isNotEmpty) {
      values['company.address.country'] = info.businessCountry;
    }
    if (info.taxId.isNotEmpty) values['company.tax_id'] = info.taxId;
  }

  // Bank fields
  if (info.iban.isNotEmpty) values['bank.iban'] = info.iban;
  if (info.bic.isNotEmpty) values['bank.bic'] = info.bic;
  if (info.accountNumber.isNotEmpty) {
    values['bank.account_number'] = info.accountNumber;
  }
  if (info.routingNumber.isNotEmpty) {
    values['bank.routing_number'] = info.routingNumber;
  }
  if (info.accountHolder.isNotEmpty) {
    values['bank.account_holder'] = info.accountHolder;
  }
  if (info.bankCountry.isNotEmpty) {
    values['bank.bank_country'] = info.bankCountry;
  }

  // Activity sector and business role (top-level keys)
  values['activity_sector'] = info.activitySector.isNotEmpty
      ? info.activitySector
      : '8999';
  if (info.roleInCompany.isNotEmpty) {
    values['business_role'] = info.roleInCompany;
  }

  // Extra fields from the entity
  for (final entry in info.extraFields.entries) {
    values[entry.key] = entry.value;
  }

  return PaymentInfoFormData(
    isBusiness: isBusiness,
    country: info.country,
    values: values,
    extraFields: Map<String, String>.from(info.extraFields),
    // Legacy flat fields (for backward compatibility)
    firstName: info.firstName,
    lastName: info.lastName,
    dateOfBirth: info.dateOfBirth,
    nationality: info.nationality,
    address: info.address,
    city: info.city,
    postalCode: info.postalCode,
    phone: info.phone,
    activitySector: info.activitySector.isNotEmpty ? info.activitySector : '8999',
    businessRole: _parseBusinessRole(info.roleInCompany),
    businessName: info.businessName,
    businessAddress: info.businessAddress,
    businessCity: info.businessCity,
    businessPostalCode: info.businessPostalCode,
    businessCountry: info.businessCountry,
    taxId: info.taxId,
    vatNumber: info.vatNumber,
    isSelfRepresentative: info.isSelfRepresentative,
    isSelfDirector: info.isSelfDirector,
    noMajorOwners: info.noMajorOwners,
    isSelfExecutive: info.isSelfExecutive,
    businessPersons: info.businessPersons
        .map((bp) => BusinessPerson(
              role: bp.role,
              firstName: bp.firstName,
              lastName: bp.lastName,
              dateOfBirth: bp.dateOfBirth,
              email: bp.email,
              phone: bp.phone,
              address: bp.address,
              city: bp.city,
              postalCode: bp.postalCode,
              title: bp.title,
            ))
        .toList(),
    bankMode: hasIban ? BankAccountMode.iban : BankAccountMode.local,
    iban: info.iban,
    bic: info.bic,
    accountNumber: info.accountNumber,
    routingNumber: info.routingNumber,
    accountHolder: info.accountHolder,
    bankCountry: info.bankCountry,
  );
}

/// Convert path-keyed [values] back to a flat JSON map for the save API.
///
/// Mirrors the web's `valuesToFlatData` function.
Map<String, dynamic> valuesToFlatData(
  PaymentInfoFormData data,
  List<FieldSection>? sections, {
  required String email,
}) {
  final v = data.values;
  final prefix = data.isBusiness ? 'representative' : 'individual';

  // Collect extra fields from sections
  final extraFields = <String, String>{...data.extraFields};
  if (sections != null) {
    for (final section in sections) {
      for (final field in section.fields) {
        if (field.isExtra && (v[field.key] ?? '').isNotEmpty) {
          extraFields[field.key] = v[field.key]!;
        }
      }
    }
  }

  final json = <String, dynamic>{
    'email': email,
    'first_name': v['$prefix.first_name'] ?? data.firstName,
    'last_name': v['$prefix.last_name'] ?? data.lastName,
    'date_of_birth': v['$prefix.dob'] ?? data.dateOfBirth,
    'nationality': v['$prefix.nationality'] ?? data.nationality.ifEmpty(data.country),
    'address': v['$prefix.address.line1'] ?? data.address,
    'city': v['$prefix.address.city'] ?? data.city,
    'postal_code': v['$prefix.address.postal_code'] ?? data.postalCode,
    'phone': v['$prefix.phone'] ?? data.phone,
    'activity_sector': v['activity_sector'] ?? data.activitySector,
    'is_business': data.isBusiness,
    'business_name': v['company.name'] ?? data.businessName,
    'business_address': v['company.address.line1'] ?? data.businessAddress,
    'business_city': v['company.address.city'] ?? data.businessCity,
    'business_postal_code':
        v['company.address.postal_code'] ?? data.businessPostalCode,
    'business_country': v['company.address.country'] ?? data.businessCountry,
    'tax_id': v['company.tax_id'] ?? data.taxId,
    'vat_number': data.vatNumber,
    'role_in_company': v['business_role'] ?? data.businessRole?.name ?? '',
    'is_self_representative': data.isSelfRepresentative,
    'is_self_director': data.isSelfDirector,
    'no_major_owners': data.noMajorOwners,
    'is_self_executive': data.isSelfExecutive,
    'iban': v['bank.iban'] ?? data.iban,
    'bic': v['bank.bic'] ?? data.bic,
    'account_number': v['bank.account_number'] ?? data.accountNumber,
    'routing_number': v['bank.routing_number'] ?? data.routingNumber,
    'account_holder': v['bank.account_holder'] ?? data.accountHolder,
    'bank_country': v['bank.bank_country'] ?? data.bankCountry,
    'country': data.country,
    'extra_fields': extraFields,
  };

  if (data.businessPersons.isNotEmpty) {
    json['business_persons'] =
        data.businessPersons.map((p) => p.toJson()).toList();
  }

  return json;
}

/// Check if all required fields from the given [sections] are filled.
bool isFormValid(PaymentInfoFormData data, List<FieldSection> sections) {
  for (final section in sections) {
    for (final field in section.fields) {
      if (field.type == 'document_upload') continue;
      if (field.required && (data.values[field.key] ?? '').trim().isEmpty) {
        return false;
      }
    }
  }
  return sections.isNotEmpty;
}

BusinessRole? _parseBusinessRole(String value) {
  const mapping = {
    'owner': BusinessRole.owner,
    'ceo': BusinessRole.ceo,
    'director': BusinessRole.director,
    'partner': BusinessRole.partner,
    'other': BusinessRole.other,
  };
  return mapping[value.toLowerCase()];
}

extension on String {
  String ifEmpty(String fallback) => isEmpty ? fallback : this;
}

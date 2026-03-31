/// Represents the bank account input mode.
enum BankAccountMode { iban, local }

/// Represents the business role of a legal representative.
enum BusinessRole { owner, ceo, director, partner, other }

/// Represents a business person (representative, director, owner, executive).
class BusinessPerson {
  const BusinessPerson({
    this.role = 'director',
    this.firstName = '',
    this.lastName = '',
    this.dateOfBirth = '',
    this.email = '',
    this.phone = '',
    this.address = '',
    this.city = '',
    this.postalCode = '',
    this.title = '',
  });

  final String role;
  final String firstName;
  final String lastName;
  final String dateOfBirth;
  final String email;
  final String phone;
  final String address;
  final String city;
  final String postalCode;
  final String title;

  BusinessPerson copyWith({
    String? role,
    String? firstName,
    String? lastName,
    String? dateOfBirth,
    String? email,
    String? phone,
    String? address,
    String? city,
    String? postalCode,
    String? title,
  }) {
    return BusinessPerson(
      role: role ?? this.role,
      firstName: firstName ?? this.firstName,
      lastName: lastName ?? this.lastName,
      dateOfBirth: dateOfBirth ?? this.dateOfBirth,
      email: email ?? this.email,
      phone: phone ?? this.phone,
      address: address ?? this.address,
      city: city ?? this.city,
      postalCode: postalCode ?? this.postalCode,
      title: title ?? this.title,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'role': role,
      'first_name': firstName,
      'last_name': lastName,
      'date_of_birth': dateOfBirth,
      'email': email,
      'phone': phone,
      'address': address,
      'city': city,
      'postal_code': postalCode,
      'title': title,
    };
  }
}

/// Form data for the payment information page.
///
/// This is a pure data class with no external dependencies. Used by the
/// presentation layer to hold form state.
class PaymentInfoFormData {
  const PaymentInfoFormData({
    this.isBusiness = false,
    this.firstName = '',
    this.lastName = '',
    this.dateOfBirth = '',
    this.nationality = '',
    this.address = '',
    this.city = '',
    this.postalCode = '',
    this.phone = '',
    this.activitySector = '8999',
    this.businessRole,
    this.businessName = '',
    this.businessAddress = '',
    this.businessCity = '',
    this.businessPostalCode = '',
    this.businessCountry = '',
    this.taxId = '',
    this.vatNumber = '',
    this.isSelfRepresentative = true,
    this.isSelfDirector = true,
    this.noMajorOwners = true,
    this.isSelfExecutive = true,
    this.businessPersons = const [],
    this.bankMode = BankAccountMode.iban,
    this.iban = '',
    this.bic = '',
    this.accountNumber = '',
    this.routingNumber = '',
    this.accountHolder = '',
    this.bankCountry = '',
    this.country = '',
    this.extraFields = const {},
  });

  final bool isBusiness;
  final String firstName;
  final String lastName;
  final String dateOfBirth;
  final String nationality;
  final String address;
  final String city;
  final String postalCode;
  final String phone;
  final String activitySector;
  final BusinessRole? businessRole;
  final String businessName;
  final String businessAddress;
  final String businessCity;
  final String businessPostalCode;
  final String businessCountry;
  final String taxId;
  final String vatNumber;
  final bool isSelfRepresentative;
  final bool isSelfDirector;
  final bool noMajorOwners;
  final bool isSelfExecutive;
  final List<BusinessPerson> businessPersons;
  final BankAccountMode bankMode;
  final String iban;
  final String bic;
  final String accountNumber;
  final String routingNumber;
  final String accountHolder;
  final String bankCountry;
  final String country;
  final Map<String, String> extraFields;

  PaymentInfoFormData copyWith({
    bool? isBusiness,
    String? firstName,
    String? lastName,
    String? dateOfBirth,
    String? nationality,
    String? address,
    String? city,
    String? postalCode,
    String? phone,
    String? activitySector,
    BusinessRole? businessRole,
    String? businessName,
    String? businessAddress,
    String? businessCity,
    String? businessPostalCode,
    String? businessCountry,
    String? taxId,
    String? vatNumber,
    bool? isSelfRepresentative,
    bool? isSelfDirector,
    bool? noMajorOwners,
    bool? isSelfExecutive,
    List<BusinessPerson>? businessPersons,
    BankAccountMode? bankMode,
    String? iban,
    String? bic,
    String? accountNumber,
    String? routingNumber,
    String? accountHolder,
    String? bankCountry,
    String? country,
    Map<String, String>? extraFields,
  }) {
    return PaymentInfoFormData(
      isBusiness: isBusiness ?? this.isBusiness,
      firstName: firstName ?? this.firstName,
      lastName: lastName ?? this.lastName,
      dateOfBirth: dateOfBirth ?? this.dateOfBirth,
      nationality: nationality ?? this.nationality,
      address: address ?? this.address,
      city: city ?? this.city,
      postalCode: postalCode ?? this.postalCode,
      phone: phone ?? this.phone,
      activitySector: activitySector ?? this.activitySector,
      businessRole: businessRole ?? this.businessRole,
      businessName: businessName ?? this.businessName,
      businessAddress: businessAddress ?? this.businessAddress,
      businessCity: businessCity ?? this.businessCity,
      businessPostalCode: businessPostalCode ?? this.businessPostalCode,
      businessCountry: businessCountry ?? this.businessCountry,
      taxId: taxId ?? this.taxId,
      vatNumber: vatNumber ?? this.vatNumber,
      isSelfRepresentative:
          isSelfRepresentative ?? this.isSelfRepresentative,
      isSelfDirector: isSelfDirector ?? this.isSelfDirector,
      noMajorOwners: noMajorOwners ?? this.noMajorOwners,
      isSelfExecutive: isSelfExecutive ?? this.isSelfExecutive,
      businessPersons: businessPersons ?? this.businessPersons,
      bankMode: bankMode ?? this.bankMode,
      iban: iban ?? this.iban,
      bic: bic ?? this.bic,
      accountNumber: accountNumber ?? this.accountNumber,
      routingNumber: routingNumber ?? this.routingNumber,
      accountHolder: accountHolder ?? this.accountHolder,
      bankCountry: bankCountry ?? this.bankCountry,
      country: country ?? this.country,
      extraFields: extraFields ?? this.extraFields,
    );
  }
}

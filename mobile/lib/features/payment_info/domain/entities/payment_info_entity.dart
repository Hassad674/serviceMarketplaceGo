/// Domain entity for a user's payment information.
class PaymentInfo {
  final String id;
  final String userId;
  final String firstName;
  final String lastName;
  final String dateOfBirth;
  final String nationality;
  final String address;
  final String city;
  final String postalCode;
  final String phone;
  final String activitySector;
  final bool isBusiness;
  final String businessName;
  final String businessAddress;
  final String businessCity;
  final String businessPostalCode;
  final String businessCountry;
  final String taxId;
  final String vatNumber;
  final String roleInCompany;
  final bool isSelfRepresentative;
  final bool isSelfDirector;
  final bool noMajorOwners;
  final bool isSelfExecutive;
  final String iban;
  final String bic;
  final String accountNumber;
  final String routingNumber;
  final String accountHolder;
  final String bankCountry;
  final String stripeAccountId;
  final bool stripeVerified;
  final DateTime createdAt;
  final DateTime updatedAt;

  const PaymentInfo({
    required this.id,
    required this.userId,
    required this.firstName,
    required this.lastName,
    required this.dateOfBirth,
    required this.nationality,
    required this.address,
    required this.city,
    required this.postalCode,
    this.phone = '',
    this.activitySector = '8999',
    this.isBusiness = false,
    this.businessName = '',
    this.businessAddress = '',
    this.businessCity = '',
    this.businessPostalCode = '',
    this.businessCountry = '',
    this.taxId = '',
    this.vatNumber = '',
    this.roleInCompany = '',
    this.isSelfRepresentative = true,
    this.isSelfDirector = true,
    this.noMajorOwners = true,
    this.isSelfExecutive = true,
    this.iban = '',
    this.bic = '',
    this.accountNumber = '',
    this.routingNumber = '',
    required this.accountHolder,
    this.bankCountry = '',
    this.stripeAccountId = '',
    this.stripeVerified = false,
    required this.createdAt,
    required this.updatedAt,
  });

  factory PaymentInfo.fromJson(Map<String, dynamic> json) {
    return PaymentInfo(
      id: json['id'] as String,
      userId: json['user_id'] as String,
      firstName: json['first_name'] as String,
      lastName: json['last_name'] as String,
      dateOfBirth: json['date_of_birth'] as String,
      nationality: json['nationality'] as String,
      address: json['address'] as String,
      city: json['city'] as String,
      postalCode: json['postal_code'] as String,
      phone: json['phone'] as String? ?? '',
      activitySector: json['activity_sector'] as String? ?? '8999',
      isBusiness: json['is_business'] as bool? ?? false,
      businessName: json['business_name'] as String? ?? '',
      businessAddress: json['business_address'] as String? ?? '',
      businessCity: json['business_city'] as String? ?? '',
      businessPostalCode: json['business_postal_code'] as String? ?? '',
      businessCountry: json['business_country'] as String? ?? '',
      taxId: json['tax_id'] as String? ?? '',
      vatNumber: json['vat_number'] as String? ?? '',
      roleInCompany: json['role_in_company'] as String? ?? '',
      isSelfRepresentative:
          json['is_self_representative'] as bool? ?? true,
      isSelfDirector: json['is_self_director'] as bool? ?? true,
      noMajorOwners: json['no_major_owners'] as bool? ?? true,
      isSelfExecutive: json['is_self_executive'] as bool? ?? true,
      iban: json['iban'] as String? ?? '',
      bic: json['bic'] as String? ?? '',
      accountNumber: json['account_number'] as String? ?? '',
      routingNumber: json['routing_number'] as String? ?? '',
      accountHolder: json['account_holder'] as String,
      bankCountry: json['bank_country'] as String? ?? '',
      stripeAccountId: json['stripe_account_id'] as String? ?? '',
      stripeVerified: json['stripe_verified'] as bool? ?? false,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }
}

/// Status of payment info completeness.
class PaymentInfoStatus {
  final bool complete;

  const PaymentInfoStatus({required this.complete});

  factory PaymentInfoStatus.fromJson(Map<String, dynamic> json) {
    return PaymentInfoStatus(
      complete: json['complete'] as bool,
    );
  }
}

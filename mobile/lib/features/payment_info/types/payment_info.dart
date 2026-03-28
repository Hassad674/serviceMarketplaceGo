/// Represents the bank account input mode.
enum BankAccountMode { iban, local }

/// Represents the business role of a legal representative.
enum BusinessRole { owner, ceo, director, partner, other }

/// Form data for the payment information page.
///
/// This is a pure data class with no external dependencies. Used by the
/// presentation layer to hold form state until the backend endpoint exists.
class PaymentInfoFormData {
  const PaymentInfoFormData({
    this.isBusiness = false,
    this.firstName = '',
    this.lastName = '',
    this.dateOfBirth = '',
    this.email = '',
    this.country = '',
    this.address = '',
    this.city = '',
    this.postalCode = '',
    this.businessRole,
    this.businessName = '',
    this.businessAddress = '',
    this.businessCity = '',
    this.businessPostalCode = '',
    this.taxId = '',
    this.vatNumber = '',
    this.bankMode = BankAccountMode.iban,
    this.iban = '',
    this.accountNumber = '',
    this.routingNumber = '',
    this.accountHolder = '',
  });

  final bool isBusiness;
  final String firstName;
  final String lastName;
  final String dateOfBirth;
  final String email;
  final String country;
  final String address;
  final String city;
  final String postalCode;
  final BusinessRole? businessRole;
  final String businessName;
  final String businessAddress;
  final String businessCity;
  final String businessPostalCode;
  final String taxId;
  final String vatNumber;
  final BankAccountMode bankMode;
  final String iban;
  final String accountNumber;
  final String routingNumber;
  final String accountHolder;

  PaymentInfoFormData copyWith({
    bool? isBusiness,
    String? firstName,
    String? lastName,
    String? dateOfBirth,
    String? email,
    String? country,
    String? address,
    String? city,
    String? postalCode,
    BusinessRole? businessRole,
    String? businessName,
    String? businessAddress,
    String? businessCity,
    String? businessPostalCode,
    String? taxId,
    String? vatNumber,
    BankAccountMode? bankMode,
    String? iban,
    String? accountNumber,
    String? routingNumber,
    String? accountHolder,
  }) {
    return PaymentInfoFormData(
      isBusiness: isBusiness ?? this.isBusiness,
      firstName: firstName ?? this.firstName,
      lastName: lastName ?? this.lastName,
      dateOfBirth: dateOfBirth ?? this.dateOfBirth,
      email: email ?? this.email,
      country: country ?? this.country,
      address: address ?? this.address,
      city: city ?? this.city,
      postalCode: postalCode ?? this.postalCode,
      businessRole: businessRole ?? this.businessRole,
      businessName: businessName ?? this.businessName,
      businessAddress: businessAddress ?? this.businessAddress,
      businessCity: businessCity ?? this.businessCity,
      businessPostalCode: businessPostalCode ?? this.businessPostalCode,
      taxId: taxId ?? this.taxId,
      vatNumber: vatNumber ?? this.vatNumber,
      bankMode: bankMode ?? this.bankMode,
      iban: iban ?? this.iban,
      accountNumber: accountNumber ?? this.accountNumber,
      routingNumber: routingNumber ?? this.routingNumber,
      accountHolder: accountHolder ?? this.accountHolder,
    );
  }
}

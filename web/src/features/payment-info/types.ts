export type BusinessRole = "owner" | "ceo" | "director" | "partner" | "other"

export type BankAccountMode = "iban" | "local"

export type PaymentInfoFormData = {
  isBusiness: boolean
  firstName: string
  lastName: string
  dateOfBirth: string
  nationality: string
  address: string
  city: string
  postalCode: string
  businessRole: BusinessRole | ""
  businessName: string
  businessAddress: string
  businessCity: string
  businessPostalCode: string
  businessCountry: string
  taxId: string
  vatNumber: string
  phone: string
  activitySector: string
  bankMode: BankAccountMode
  iban: string
  bic: string
  accountNumber: string
  routingNumber: string
  accountHolder: string
  bankCountry: string
}

export const INITIAL_FORM_DATA: PaymentInfoFormData = {
  isBusiness: false,
  firstName: "",
  lastName: "",
  dateOfBirth: "",
  nationality: "",
  address: "",
  city: "",
  postalCode: "",
  businessRole: "",
  businessName: "",
  businessAddress: "",
  businessCity: "",
  businessPostalCode: "",
  businessCountry: "",
  taxId: "",
  vatNumber: "",
  phone: "",
  activitySector: "8999",
  bankMode: "iban",
  iban: "",
  bic: "",
  accountNumber: "",
  routingNumber: "",
  accountHolder: "",
  bankCountry: "",
}

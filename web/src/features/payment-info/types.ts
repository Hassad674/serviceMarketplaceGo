export type BusinessRole = "owner" | "ceo" | "director" | "partner" | "other"

export type BankAccountMode = "iban" | "local"

export type BusinessPersonData = {
  role: string
  firstName: string
  lastName: string
  dateOfBirth: string
  email: string
  phone: string
  address: string
  city: string
  postalCode: string
  title: string
}

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
  isSelfRepresentative: boolean
  isSelfDirector: boolean
  noMajorOwners: boolean
  isSelfExecutive: boolean
  businessPersons: BusinessPersonData[]
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
  isSelfRepresentative: true,
  isSelfDirector: true,
  noMajorOwners: true,
  isSelfExecutive: true,
  businessPersons: [],
  bankMode: "iban",
  iban: "",
  bic: "",
  accountNumber: "",
  routingNumber: "",
  accountHolder: "",
  bankCountry: "",
}

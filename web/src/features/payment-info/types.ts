export type BusinessRole = "owner" | "ceo" | "director" | "partner" | "other"

export type BankAccountMode = "iban" | "local"

export type PaymentInfoFormData = {
  isBusiness: boolean
  firstName: string
  lastName: string
  dateOfBirth: string
  email: string
  country: string
  address: string
  city: string
  postalCode: string
  businessRole: BusinessRole | ""
  businessName: string
  businessAddress: string
  businessCity: string
  businessPostalCode: string
  taxId: string
  vatNumber: string
  bankMode: BankAccountMode
  iban: string
  accountNumber: string
  routingNumber: string
  accountHolder: string
}

export const INITIAL_FORM_DATA: PaymentInfoFormData = {
  isBusiness: false,
  firstName: "",
  lastName: "",
  dateOfBirth: "",
  email: "",
  country: "",
  address: "",
  city: "",
  postalCode: "",
  businessRole: "",
  businessName: "",
  businessAddress: "",
  businessCity: "",
  businessPostalCode: "",
  taxId: "",
  vatNumber: "",
  bankMode: "iban",
  iban: "",
  accountNumber: "",
  routingNumber: "",
  accountHolder: "",
}

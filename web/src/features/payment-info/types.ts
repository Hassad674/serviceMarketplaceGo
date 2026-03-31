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
  country: string
  /** Path-keyed values: "individual.first_name" -> "Jean" */
  values: Record<string, string>
  // Legacy flat fields kept for save compatibility
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
  extraFields: Record<string, string>
}

export const INITIAL_FORM_DATA: PaymentInfoFormData = {
  isBusiness: false,
  country: "",
  values: {},
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
  extraFields: {},
}

/** Activity sector options for the MCC dropdown. */
export const ACTIVITY_SECTORS = [
  { mcc: "7372", labelKey: "sectorDev" },
  { mcc: "7333", labelKey: "sectorDesign" },
  { mcc: "7311", labelKey: "sectorMarketing" },
  { mcc: "7392", labelKey: "sectorConsulting" },
  { mcc: "7339", labelKey: "sectorAdmin" },
  { mcc: "7221", labelKey: "sectorPhoto" },
  { mcc: "7338", labelKey: "sectorWriting" },
  { mcc: "8299", labelKey: "sectorTraining" },
  { mcc: "8931", labelKey: "sectorAccounting" },
  { mcc: "8911", labelKey: "sectorEngineering" },
  { mcc: "8111", labelKey: "sectorLegal" },
  { mcc: "8099", labelKey: "sectorHealth" },
  { mcc: "8999", labelKey: "sectorOther" },
] as const

/** Business role options. */
export const BUSINESS_ROLES: { value: BusinessRole; labelKey: string }[] = [
  { value: "owner", labelKey: "roleOwner" },
  { value: "ceo", labelKey: "roleCeo" },
  { value: "director", labelKey: "roleDirector" },
  { value: "partner", labelKey: "rolePartner" },
  { value: "other", labelKey: "roleOther" },
]

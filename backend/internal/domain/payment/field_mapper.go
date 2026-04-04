package payment

import "strings"

// FieldMapping describes how a Stripe field path maps to our storage.
type FieldMapping struct {
	DBField string // column name or extra_fields key
	IsExtra bool   // true if stored in extra_fields JSONB
}

// knownFieldMappings maps Stripe field paths to our DB columns.
// Fields not in this map go to extra_fields with their short key.
var knownFieldMappings = map[string]FieldMapping{
	"individual.first_name":          {DBField: "first_name", IsExtra: false},
	"individual.last_name":           {DBField: "last_name", IsExtra: false},
	"individual.dob.day":             {DBField: "date_of_birth", IsExtra: false},
	"individual.dob.month":           {DBField: "date_of_birth", IsExtra: false},
	"individual.dob.year":            {DBField: "date_of_birth", IsExtra: false},
	"individual.email":               {DBField: "email", IsExtra: false},
	"individual.phone":               {DBField: "phone", IsExtra: false},
	"individual.address.line1":       {DBField: "address", IsExtra: false},
	"individual.address.city":        {DBField: "city", IsExtra: false},
	"individual.address.postal_code": {DBField: "postal_code", IsExtra: false},
	"individual.address.state":       {DBField: "state", IsExtra: true},

	"individual.id_number":          {DBField: "id_number", IsExtra: true},
	"individual.ssn_last_4":         {DBField: "ssn_last_4", IsExtra: true},
	"individual.political_exposure": {DBField: "political_exposure", IsExtra: true},

	"individual.first_name_kana":  {DBField: "first_name_kana", IsExtra: true},
	"individual.last_name_kana":   {DBField: "last_name_kana", IsExtra: true},
	"individual.first_name_kanji": {DBField: "first_name_kanji", IsExtra: true},
	"individual.last_name_kanji":  {DBField: "last_name_kanji", IsExtra: true},

	"individual.address.kana.line1":       {DBField: "address_kana_line1", IsExtra: true},
	"individual.address.kana.city":        {DBField: "address_kana_city", IsExtra: true},
	"individual.address.kana.town":        {DBField: "address_kana_town", IsExtra: true},
	"individual.address.kana.postal_code": {DBField: "address_kana_postal_code", IsExtra: true},
	"individual.address.kana.state":       {DBField: "address_kana_state", IsExtra: true},

	"individual.address.kanji.line1":       {DBField: "address_kanji_line1", IsExtra: true},
	"individual.address.kanji.city":        {DBField: "address_kanji_city", IsExtra: true},
	"individual.address.kanji.town":        {DBField: "address_kanji_town", IsExtra: true},
	"individual.address.kanji.postal_code": {DBField: "address_kanji_postal_code", IsExtra: true},
	"individual.address.kanji.state":       {DBField: "address_kanji_state", IsExtra: true},

	"representative.first_name":          {DBField: "first_name", IsExtra: false},
	"representative.last_name":           {DBField: "last_name", IsExtra: false},
	"representative.dob.day":             {DBField: "date_of_birth", IsExtra: false},
	"representative.dob.month":           {DBField: "date_of_birth", IsExtra: false},
	"representative.dob.year":            {DBField: "date_of_birth", IsExtra: false},
	"representative.email":               {DBField: "email", IsExtra: false},
	"representative.phone":               {DBField: "phone", IsExtra: false},
	"representative.address.line1":       {DBField: "address", IsExtra: false},
	"representative.address.city":        {DBField: "city", IsExtra: false},
	"representative.address.postal_code": {DBField: "postal_code", IsExtra: false},
	"representative.address.state":       {DBField: "state", IsExtra: true},

	"company.name":                {DBField: "business_name", IsExtra: false},
	"company.phone":               {DBField: "company.phone", IsExtra: true},
	"company.address.line1":       {DBField: "business_address", IsExtra: false},
	"company.address.city":        {DBField: "business_city", IsExtra: false},
	"company.address.postal_code": {DBField: "business_postal_code", IsExtra: false},
	"company.address.state":       {DBField: "business_state", IsExtra: true},
	"company.tax_id":              {DBField: "tax_id", IsExtra: false},
}

// MapStripeField maps a Stripe field path to our storage location.
func MapStripeField(path string) (dbField string, isExtra bool) {
	if m, ok := knownFieldMappings[path]; ok {
		return m.DBField, m.IsExtra
	}
	parts := strings.Split(path, ".")
	key := parts[len(parts)-1]
	return key, true
}

// IsDocumentUploadField returns true for any path that represents a file/document
// upload requirement from Stripe. These should render as upload zones, not text inputs.
func IsDocumentUploadField(path string) bool {
	if strings.Contains(path, "verification.document") ||
		strings.Contains(path, "verification.additional_document") ||
		strings.Contains(path, "verification.proof_of_liveness") ||
		strings.Contains(path, "documents.") ||
		strings.HasSuffix(path, ".files") {
		return true
	}
	return false
}

// DocumentCategoryFromPath returns "individual" or "company" based on the entity
// prefix of a document upload path.
func DocumentCategoryFromPath(path string) string {
	entity := EntityFromPath(path)
	if entity == "company" || entity == "documents" {
		return "company"
	}
	return "individual"
}

// DocumentTypeFromPath derives a document_type label from a Stripe document path.
func DocumentTypeFromPath(path string) string {
	if strings.Contains(path, "proof_of_liveness") {
		return "proof_of_liveness"
	}
	if strings.Contains(path, "additional_document") {
		return "additional_document"
	}
	if strings.Contains(path, "company_authorization") {
		return "company_authorization"
	}
	if strings.Contains(path, "passport") {
		return "passport"
	}
	if strings.Contains(path, "bank_account_ownership_verification") {
		return "bank_account_ownership"
	}
	if strings.Contains(path, "proof_of_registration") {
		return "proof_of_registration"
	}
	return "document"
}

// personRolePrefixes maps Stripe field path prefixes to person roles.
var personRolePrefixes = []struct {
	prefix string
	role   string
}{
	{prefix: "directors.", role: "director"},
	{prefix: "executives.", role: "executive"},
	{prefix: "owners.", role: "owner"},
	{prefix: "representative.", role: "representative"},
}

// RequiredPersonRoles extracts which person roles are needed.
func RequiredPersonRoles(companyMinimum []string) []string {
	seen := make(map[string]bool)
	for _, field := range companyMinimum {
		for _, pr := range personRolePrefixes {
			if strings.HasPrefix(field, pr.prefix) && !seen[pr.role] {
				seen[pr.role] = true
			}
		}
	}
	roles := make([]string, 0, len(seen))
	for role := range seen {
		roles = append(roles, role)
	}
	return roles
}

// autoHandledPrefixes are Stripe paths we handle internally.
var autoHandledPrefixes = []string{
	"tos_acceptance",
	"business_type",
	"business_profile.",
	"external_account",
	"settings.",
	"company.ownership_declaration.",
	"company.directorship_declaration.",
}

// IsAutoHandled returns true for paths handled internally.
func IsAutoHandled(path string) bool {
	for _, prefix := range autoHandledPrefixes {
		if path == prefix || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	if strings.HasSuffix(path, "_provided") {
		return true
	}
	return false
}

// IsDOBComponent returns true for dob.day, dob.month, dob.year.
func IsDOBComponent(path string) bool {
	return strings.HasSuffix(path, ".dob.day") ||
		strings.HasSuffix(path, ".dob.month") ||
		strings.HasSuffix(path, ".dob.year")
}

// IsRegistrationDateComponent returns true for registration_date.day/month/year.
func IsRegistrationDateComponent(path string) bool {
	return strings.HasSuffix(path, ".registration_date.day") ||
		strings.HasSuffix(path, ".registration_date.month") ||
		strings.HasSuffix(path, ".registration_date.year")
}

// EntityFromPath extracts the entity prefix from a Stripe path.
// Normalizes dynamic person IDs (person_XXXX) to "person".
func EntityFromPath(path string) string {
	idx := strings.IndexByte(path, '.')
	if idx < 0 {
		return path
	}
	entity := path[:idx]
	// Normalize Stripe person IDs (person_1TIWrv...) to generic "person"
	if strings.HasPrefix(entity, "person_") {
		return "person"
	}
	return entity
}

// sectionTitleKeys maps entity names to i18n title keys.
var sectionTitleKeys = map[string]string{
	"individual":     "personalInfo",
	"company":        "companyInfo",
	"representative": "legalRepresentative",
	"directors":      "directors",
	"owners":         "owners",
	"executives":     "executives",
	"authorizer":     "authorizer",
	"documents":      "companyDocuments",
	"person":         "personInfo",
}

// SectionTitleKey maps an entity to its i18n title key.
func SectionTitleKey(entity string) string {
	if key, ok := sectionTitleKeys[entity]; ok {
		return key
	}
	return entity
}

// FieldInputType returns the input type for a Stripe field path.
func FieldInputType(path string) string {
	if IsDocumentUploadField(path) {
		return "document_upload"
	}
	terminal := terminalSegment(path)
	switch terminal {
	case "email":
		return "email"
	case "phone":
		return "phone"
	case "dob", "registration_date":
		return "date"
	case "nationality", "country", "political_exposure",
		"structure", "gender", "vat_registration_status":
		return "select"
	case "executive":
		return "select"
	default:
		return "text"
	}
}

// fieldLabelKeys maps terminal field names to i18n label keys.
var fieldLabelKeys = map[string]string{
	"first_name":          "firstName",
	"last_name":           "lastName",
	"dob":                 "dateOfBirth",
	"email":               "email",
	"phone":               "phone",
	"line1":               "address",
	"line2":               "addressLine2",
	"city":                "city",
	"postal_code":         "postalCode",
	"state":               "stateProvince",
	"country":             "country",
	"nationality":         "nationality",
	"name":                "businessName",
	"tax_id":              "taxId",
	"id_number":           "idNumber",
	"id_number_secondary": "idNumberSecondary",
	"ssn_last_4":          "ssnLast4",
	"first_name_kana":     "firstNameKana",
	"last_name_kana":      "lastNameKana",
	"first_name_kanji":    "firstNameKanji",
	"last_name_kanji":     "lastNameKanji",
	"political_exposure":  "politicalExposure",
	"town":                "town",
	"full_name_aliases":   "fullNameAliases",
	"gender":              "gender",
	"percent_ownership":   "percentOwnership",
	"structure":           "companyStructure",
	"registration_number": "registrationNumber",
	"registration_date":   "registrationDate",
	"name_kana":           "businessNameKana",
	"name_kanji":          "businessNameKanji",
	"vat_id":              "vatId",
	"executive":           "isExecutive",
	"business_vat_id_number":         "businessVatIdNumber",
	"vat_registration_status":        "vatRegistrationStatus",
	"ownership_exemption_reason":     "ownershipExemptionReason",
	"business_cross_border_transaction_classifications": "crossBorderClassifications",
}

// companyLabelOverrides maps company-specific fields to distinct i18n labels.
var companyLabelOverrides = map[string]string{
	"company.address.line1":       "businessAddress",
	"company.address.city":        "businessCity",
	"company.address.postal_code": "businessPostalCode",
	"company.address.state":       "businessState",
	"company.address.country":     "businessCountry",
	"company.phone":               "phone",
}

// FieldLabelKey returns the i18n label key for a Stripe field path.
func FieldLabelKey(path string) string {
	// Handle document upload fields with specific labels per document type.
	// Order matters: check more specific paths before generic ones.
	if IsDocumentUploadField(path) {
		if strings.Contains(path, "proof_of_liveness") {
			return "docProofOfLiveness"
		}
		if strings.Contains(path, "bank_account_ownership_verification") {
			return "docBankOwnership"
		}
		if strings.Contains(path, "proof_of_registration") {
			return "docProofOfRegistration"
		}
		if strings.Contains(path, "additional_document") {
			return "docAdditionalDocument"
		}
		if strings.HasPrefix(path, "company.") && strings.Contains(path, "verification.document") {
			return "docCompanyDocument"
		}
		if strings.Contains(path, "verification.document") {
			return "docVerificationDocument"
		}
		return "docVerificationDocument"
	}

	// Check entity-specific overrides first
	if key, ok := companyLabelOverrides[path]; ok {
		return key
	}

	// Handle relationship sub-fields
	if strings.Contains(path, "relationship.") {
		terminal := terminalSegment(path)
		switch terminal {
		case "title":
			return "roleTitle"
		case "percent_ownership":
			return "percentOwnership"
		case "executive":
			return "isExecutive"
		}
	}

	// Handle registered_address fields — map to registered address labels
	if strings.Contains(path, "registered_address.") {
		terminal := terminalSegment(path)
		switch terminal {
		case "line1":
			return "registeredAddress"
		case "city":
			return "registeredCity"
		case "postal_code":
			return "registeredPostalCode"
		case "state":
			return "registeredState"
		}
	}

	// Handle contact_point_verification_address fields
	if strings.Contains(path, "contact_point_verification_address.") {
		terminal := terminalSegment(path)
		switch terminal {
		case "line1":
			return "contactAddress"
		case "line2":
			return "contactAddressLine2"
		case "city":
			return "contactCity"
		case "postal_code":
			return "contactPostalCode"
		case "state":
			return "contactState"
		case "town":
			return "contactTown"
		}
	}

	terminal := terminalSegment(path)
	if key, ok := fieldLabelKeys[terminal]; ok {
		return key
	}
	// Fallback: humanize the terminal segment (snake_case to camelCase)
	return humanizeTerminal(terminal)
}

// humanizeTerminal converts a snake_case terminal to a camelCase label key.
// e.g. "percent_ownership" -> "percentOwnership", "title" -> "title"
func humanizeTerminal(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) <= 1 {
		return s
	}
	result := parts[0]
	for _, p := range parts[1:] {
		if len(p) > 0 {
			result += strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return result
}

// FieldPlaceholder returns an optional placeholder string.
func FieldPlaceholder(path string) string {
	terminal := terminalSegment(path)
	placeholders := map[string]string{
		"state":      "State / Province",
		"id_number":  "National ID number",
		"ssn_last_4": "Last 4 digits of SSN",
		"phone":      "+33 6 12 34 56 78",
	}
	if p, ok := placeholders[terminal]; ok {
		return p
	}
	return ""
}

// terminalSegment returns the last segment of a dot-separated path.
func terminalSegment(path string) string {
	idx := strings.LastIndexByte(path, '.')
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}

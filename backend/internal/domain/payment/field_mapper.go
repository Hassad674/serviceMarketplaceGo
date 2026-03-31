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
	"company.phone":               {DBField: "phone", IsExtra: false},
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

// documentPrefixes maps Stripe verification paths to document types.
var documentPrefixes = map[string]string{
	"individual.verification.document":                "individual",
	"individual.verification.additional_document":     "individual",
	"company.verification.document":                   "company",
	"company.verification.additional_document":        "company",
	"representative.verification.document":            "individual",
	"representative.verification.additional_document": "individual",
}

// IsDocumentField checks if a Stripe field path is a document upload requirement.
func IsDocumentField(path string) (docType string, ok bool) {
	for prefix, dt := range documentPrefixes {
		if strings.HasPrefix(path, prefix) {
			return dt, true
		}
	}
	return "", false
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

// EntityFromPath extracts the entity prefix from a Stripe path.
func EntityFromPath(path string) string {
	idx := strings.IndexByte(path, '.')
	if idx < 0 {
		return path
	}
	return path[:idx]
}

// sectionTitleKeys maps entity names to i18n title keys.
var sectionTitleKeys = map[string]string{
	"individual":     "personalInfo",
	"company":        "companyInfo",
	"representative": "legalRepresentative",
	"directors":      "directors",
	"owners":         "owners",
	"executives":     "executives",
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
	terminal := terminalSegment(path)
	switch terminal {
	case "email":
		return "email"
	case "phone":
		return "phone"
	case "dob":
		return "date"
	case "nationality", "country", "political_exposure":
		return "select"
	default:
		return "text"
	}
}

// fieldLabelKeys maps terminal field names to i18n label keys.
var fieldLabelKeys = map[string]string{
	"first_name":        "firstName",
	"last_name":         "lastName",
	"dob":               "dateOfBirth",
	"email":             "email",
	"phone":             "phone",
	"line1":             "address",
	"city":              "city",
	"postal_code":       "postalCode",
	"state":             "stateProvince",
	"country":           "country",
	"nationality":       "nationality",
	"name":              "businessName",
	"tax_id":            "taxId",
	"id_number":         "idNumber",
	"ssn_last_4":        "ssnLast4",
	"first_name_kana":   "firstNameKana",
	"last_name_kana":    "lastNameKana",
	"first_name_kanji":  "firstNameKanji",
	"last_name_kanji":   "lastNameKanji",
	"political_exposure": "politicalExposure",
	"town":              "town",
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
	// Check entity-specific overrides first
	if key, ok := companyLabelOverrides[path]; ok {
		return key
	}
	terminal := terminalSegment(path)
	if key, ok := fieldLabelKeys[terminal]; ok {
		return key
	}
	if terminal == "title" && strings.Contains(path, "relationship") {
		return "roleTitle"
	}
	return terminal
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

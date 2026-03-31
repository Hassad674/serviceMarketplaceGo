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

	"individual.id_number":           {DBField: "id_number", IsExtra: true},
	"individual.ssn_last_4":          {DBField: "ssn_last_4", IsExtra: true},
	"individual.political_exposure":  {DBField: "political_exposure", IsExtra: true},

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

	"company.name":                  {DBField: "business_name", IsExtra: false},
	"company.phone":                 {DBField: "phone", IsExtra: false},
	"company.address.line1":         {DBField: "business_address", IsExtra: false},
	"company.address.city":          {DBField: "business_city", IsExtra: false},
	"company.address.postal_code":   {DBField: "business_postal_code", IsExtra: false},
	"company.address.state":         {DBField: "business_state", IsExtra: true},
	"company.tax_id":                {DBField: "tax_id", IsExtra: false},
}

// MapStripeField maps a Stripe field path to our storage location.
func MapStripeField(path string) (dbField string, isExtra bool) {
	if m, ok := knownFieldMappings[path]; ok {
		return m.DBField, m.IsExtra
	}
	// Unknown field: extract last segment as key, store in extra_fields
	parts := strings.Split(path, ".")
	key := parts[len(parts)-1]
	return key, true
}

// documentPrefixes are Stripe field paths that indicate document requirements.
var documentPrefixes = map[string]string{
	"individual.verification.document": "individual",
	"company.verification.document":    "company",
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

// RequiredPersonRoles extracts which person roles are needed from company minimum fields.
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

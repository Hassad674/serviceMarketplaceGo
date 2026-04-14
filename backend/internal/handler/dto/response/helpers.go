package response

// nilToEmptyStrings guarantees a non-nil slice so JSON marshaling
// yields `[]` rather than null. Shared helper used by the split-
// profile DTOs (freelance + referrer) so both JSON shapes stay
// stable for clients.
func nilToEmptyStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

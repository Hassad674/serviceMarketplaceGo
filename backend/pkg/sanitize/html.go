package sanitize

import "github.com/microcosm-cc/bluemonday"

var strict = bluemonday.StrictPolicy()

// StripHTML removes all HTML tags from the input string, returning plain text.
func StripHTML(s string) string {
	return strict.Sanitize(s)
}

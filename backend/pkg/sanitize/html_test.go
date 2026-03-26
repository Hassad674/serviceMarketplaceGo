package sanitize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripHTML_RemovesBasicTags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text unchanged",
			input:    "Hello world",
			expected: "Hello world",
		},
		{
			name:     "removes bold tags",
			input:    "<b>Hello</b> world",
			expected: "Hello world",
		},
		{
			name:     "removes script tags and content",
			input:    "Hello <script>alert('xss')</script> world",
			expected: "Hello  world",
		},
		{
			name:     "removes anchor tags",
			input:    `Click <a href="https://evil.com">here</a>`,
			expected: "Click here",
		},
		{
			name:     "removes nested tags",
			input:    "<div><p><strong>Nested</strong></p></div>",
			expected: "Nested",
		},
		{
			name:     "handles img tag",
			input:    `<img src="x" onerror="alert(1)">`,
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only tags no content",
			input:    "<div><span></span></div>",
			expected: "",
		},
		{
			name:     "preserves html entities",
			input:    "Hello &amp; world",
			expected: "Hello &amp; world",
		},
		{
			name:     "removes style tags and content",
			input:    "Hello <style>body{color:red}</style> world",
			expected: "Hello  world",
		},
		{
			name:     "removes iframe",
			input:    `<iframe src="https://evil.com"></iframe>`,
			expected: "",
		},
		{
			name:     "handles event handlers in attributes",
			input:    `<div onmouseover="alert('xss')">text</div>`,
			expected: "text",
		},
		{
			name:     "unicode content preserved",
			input:    "Bonjour <b>le monde</b> 🌍",
			expected: "Bonjour le monde 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTML(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

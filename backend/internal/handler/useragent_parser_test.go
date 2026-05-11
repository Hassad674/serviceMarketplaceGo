package handler

import "testing"

// TestParseUserAgent_Matrix locks down the realistic UA strings the
// "Sécurité" page will encounter. The table is the contract — if any
// row breaks, the Malt-style label is wrong for that device.
func TestParseUserAgent_Matrix(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		wantLbl  string
		wantBrwz string
		wantOS   string
	}{
		{
			name:     "chrome on windows desktop",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
			wantLbl:  "Ordinateur de bureau (Chrome)",
			wantBrwz: "Chrome",
			wantOS:   "Windows",
		},
		{
			name:     "edge on windows desktop",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36 Edg/118.0.2088.46",
			wantLbl:  "Ordinateur de bureau (Edge)",
			wantBrwz: "Edge",
			wantOS:   "Windows",
		},
		{
			name:     "firefox on macos desktop",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0",
			wantLbl:  "Ordinateur de bureau (Firefox)",
			wantBrwz: "Firefox",
			wantOS:   "macOS",
		},
		{
			name:     "safari on macos desktop",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			wantLbl:  "Ordinateur de bureau (Safari)",
			wantBrwz: "Safari",
			wantOS:   "macOS",
		},
		{
			name:     "chrome on linux desktop",
			ua:       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
			wantLbl:  "Ordinateur de bureau (Chrome)",
			wantBrwz: "Chrome",
			wantOS:   "Linux",
		},
		{
			name:     "safari on iphone",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantLbl:  "iPhone (Safari)",
			wantBrwz: "Safari",
			wantOS:   "iOS",
		},
		{
			name:     "chrome on iphone reports as CriOS",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/118.0.5993.92 Mobile/15E148 Safari/604.1",
			wantLbl:  "iPhone (Chrome)",
			wantBrwz: "Chrome",
			wantOS:   "iOS",
		},
		{
			name:     "safari on ipad",
			ua:       "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantLbl:  "iPad (Safari)",
			wantBrwz: "Safari",
			wantOS:   "iOS",
		},
		{
			name:     "chrome on android phone",
			ua:       "Mozilla/5.0 (Linux; Android 14; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Mobile Safari/537.36",
			wantLbl:  "Android (Chrome)",
			wantBrwz: "Chrome",
			wantOS:   "Android",
		},
		{
			name:     "firefox on android phone",
			ua:       "Mozilla/5.0 (Android 14; Mobile; rv:120.0) Gecko/120.0 Firefox/120.0",
			wantLbl:  "Android (Firefox)",
			wantBrwz: "Firefox",
			wantOS:   "Android",
		},
		{
			name:    "empty UA falls back to unknown",
			ua:      "",
			wantLbl: UnknownDeviceLabel,
		},
		{
			name:    "bot UA falls back to unknown",
			ua:      "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			wantLbl: UnknownDeviceLabel,
		},
		{
			name:    "curl falls back to unknown",
			ua:      "curl/8.4.0",
			wantLbl: UnknownDeviceLabel,
		},
		{
			name:    "completely unrecognised UA falls back to unknown",
			ua:      "totally-made-up/1.0",
			wantLbl: UnknownDeviceLabel,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ParseUserAgent(tc.ua)
			if got.Label != tc.wantLbl {
				t.Errorf("Label = %q, want %q", got.Label, tc.wantLbl)
			}
			if tc.wantBrwz != "" && got.Browser != tc.wantBrwz {
				t.Errorf("Browser = %q, want %q", got.Browser, tc.wantBrwz)
			}
			if tc.wantOS != "" && got.OS != tc.wantOS {
				t.Errorf("OS = %q, want %q", got.OS, tc.wantOS)
			}
		})
	}
}

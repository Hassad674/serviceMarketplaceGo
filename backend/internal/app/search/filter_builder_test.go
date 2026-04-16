package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// filter_builder_test.go is an exhaustive table-driven test for the
// pure filter_by builder. Every clause has both an "active" and an
// "inactive" branch so a regression on either side is caught.

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat(v float64) *float64   { return &v }
func ptrBool(v bool) *bool          { return &v }

func TestBuildFilterBy_Empty(t *testing.T) {
	assert.Equal(t, "", BuildFilterBy(FilterInput{}))
}

func TestBuildFilterBy_Availability(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"single value", []string{"available_now"}, "availability_status:[available_now]"},
		{"multi value", []string{"available_now", "available_soon"}, "availability_status:[available_now,available_soon]"},
		{"with whitespace", []string{" available_now ", "available_soon"}, "availability_status:[available_now,available_soon]"},
		{"with duplicates", []string{"available_now", "available_now"}, "availability_status:[available_now]"},
		{"empty", nil, ""},
		{"all blanks", []string{"  ", ""}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFilterBy(FilterInput{AvailabilityStatus: tt.in})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterBy_PricingRange(t *testing.T) {
	tests := []struct {
		name   string
		min    *int64
		max    *int64
		want   string
	}{
		{"both bounds", ptrInt64(50000), ptrInt64(150000), "pricing_min_amount:>=50000 && pricing_min_amount:<=150000"},
		{"only min", ptrInt64(50000), nil, "pricing_min_amount:>=50000"},
		{"only max", nil, ptrInt64(150000), "pricing_min_amount:<=150000"},
		{"neither", nil, nil, ""},
		{"zero min", ptrInt64(0), nil, "pricing_min_amount:>=0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFilterBy(FilterInput{PricingMin: tt.min, PricingMax: tt.max})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterBy_City(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Paris", "city:`Paris`"},
		{"with space", "New York", "city:`New York`"},
		{"trimmed", "  Paris  ", "city:`Paris`"},
		{"empty", "", ""},
		{"whitespace only", "   ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFilterBy(FilterInput{City: tt.in})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterBy_CountryCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"lowercase preserved", "fr", "country_code:fr"},
		{"uppercase preserved", "FR", "country_code:FR"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFilterBy(FilterInput{CountryCode: tt.in})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterBy_Geo(t *testing.T) {
	tests := []struct {
		name   string
		lat    *float64
		lng    *float64
		radius *float64
		want   string
	}{
		{"all set", ptrFloat(48.8566), ptrFloat(2.3522), ptrFloat(25), "location:(48.8566,2.3522,25 km)"},
		{"missing lat", nil, ptrFloat(2.3522), ptrFloat(25), ""},
		{"missing lng", ptrFloat(48.8566), nil, ptrFloat(25), ""},
		{"missing radius", ptrFloat(48.8566), ptrFloat(2.3522), nil, ""},
		{"zero radius", ptrFloat(48.8566), ptrFloat(2.3522), ptrFloat(0), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFilterBy(FilterInput{
				GeoLat:      tt.lat,
				GeoLng:      tt.lng,
				GeoRadiusKm: tt.radius,
			})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterBy_Languages(t *testing.T) {
	got := BuildFilterBy(FilterInput{Languages: []string{"fr", "en"}})
	assert.Equal(t, "languages_professional:[fr,en]", got)
}

func TestBuildFilterBy_Expertise(t *testing.T) {
	got := BuildFilterBy(FilterInput{ExpertiseDomains: []string{"dev", "design"}})
	assert.Equal(t, "expertise_domains:[dev,design]", got)
}

func TestBuildFilterBy_Skills(t *testing.T) {
	got := BuildFilterBy(FilterInput{Skills: []string{"react", "go"}})
	assert.Equal(t, "skills:[react,go]", got)
}

func TestBuildFilterBy_RatingMin(t *testing.T) {
	tests := []struct {
		name string
		in   *float64
		want string
	}{
		{"4 stars", ptrFloat(4), "rating_average:>=4"},
		{"4.5 stars", ptrFloat(4.5), "rating_average:>=4.5"},
		{"nil", nil, ""},
		{"zero", ptrFloat(0), ""},
		{"negative", ptrFloat(-1), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildFilterBy(FilterInput{RatingMin: tt.in})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildFilterBy_WorkMode(t *testing.T) {
	got := BuildFilterBy(FilterInput{WorkMode: []string{"remote", "hybrid"}})
	assert.Equal(t, "work_mode:[remote,hybrid]", got)
}

func TestBuildFilterBy_BooleanToggles(t *testing.T) {
	tests := []struct {
		name string
		in   FilterInput
		want string
	}{
		{"verified true", FilterInput{IsVerified: ptrBool(true)}, "is_verified:=true"},
		{"verified false", FilterInput{IsVerified: ptrBool(false)}, "is_verified:=false"},
		{"top rated", FilterInput{IsTopRated: ptrBool(true)}, "is_top_rated:=true"},
		{"negotiable", FilterInput{Negotiable: ptrBool(true)}, "pricing_negotiable:=true"},
		{"all nil", FilterInput{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, BuildFilterBy(tt.in))
		})
	}
}

func TestBuildFilterBy_CombinedFilters(t *testing.T) {
	got := BuildFilterBy(FilterInput{
		AvailabilityStatus: []string{"available_now"},
		PricingMin:         ptrInt64(40000),
		PricingMax:         ptrInt64(120000),
		City:               "Paris",
		CountryCode:        "FR",
		Languages:          []string{"fr", "en"},
		Skills:             []string{"react"},
		RatingMin:          ptrFloat(4),
		WorkMode:           []string{"remote"},
		IsVerified:         ptrBool(true),
		IsTopRated:         ptrBool(true),
	})

	want := "availability_status:[available_now]" +
		" && pricing_min_amount:>=40000 && pricing_min_amount:<=120000" +
		" && city:`Paris`" +
		" && country_code:FR" +
		" && languages_professional:[fr,en]" +
		" && skills:[react]" +
		" && rating_average:>=4" +
		" && work_mode:[remote]" +
		" && is_verified:=true" +
		" && is_top_rated:=true"
	assert.Equal(t, want, got)
}

func TestBuildFilterBy_OrderIsStable(t *testing.T) {
	// Same input produces the same output across multiple calls so
	// downstream caches (Typesense query cache, TanStack Query) hit
	// reliably.
	in := FilterInput{
		Skills:    []string{"go", "react"},
		Languages: []string{"fr"},
	}
	got1 := BuildFilterBy(in)
	got2 := BuildFilterBy(in)
	assert.Equal(t, got1, got2)
}

package consent_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/consent"
)

func TestAction_IsValid(t *testing.T) {
	cases := []struct {
		in   consent.Action
		want bool
	}{
		{consent.ActionAcceptAll, true},
		{consent.ActionRefuseAll, true},
		{consent.ActionCustom, true},
		{consent.Action(""), false},
		{consent.Action("foo"), false},
	}
	for _, c := range cases {
		if got := c.in.IsValid(); got != c.want {
			t.Errorf("IsValid(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNew_HappyPath(t *testing.T) {
	uid := uuid.New()
	entry, err := consent.New(consent.NewInput{
		UserID:        &uid,
		SessionID:     " sess-1 ",
		Categories:    []string{"analytics", "analytics", " ", "functional"},
		Action:        consent.ActionAcceptAll,
		IPAnonymized:  "203.0.x.x",
		UserAgentHash: "deadbeef",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if entry.ID == uuid.Nil {
		t.Errorf("expected fresh ID")
	}
	if entry.SessionID != "sess-1" {
		t.Errorf("session not trimmed: %q", entry.SessionID)
	}
	if len(entry.Categories) != 2 ||
		entry.Categories[0] != "analytics" ||
		entry.Categories[1] != "functional" {
		t.Errorf("categories not normalized: %v", entry.Categories)
	}
	if entry.UserID == nil || *entry.UserID != uid {
		t.Errorf("user id not preserved")
	}
	if entry.CreatedAt.IsZero() {
		t.Errorf("created_at not set")
	}
}

func TestNew_RejectsInvalidInputs(t *testing.T) {
	uid := uuid.New()
	cases := []struct {
		name string
		in   consent.NewInput
		want error
	}{
		{
			name: "invalid action",
			in: consent.NewInput{
				UserID:        &uid,
				Categories:    []string{"analytics"},
				Action:        "bogus",
				IPAnonymized:  "1.2.x.x",
				UserAgentHash: "h",
			},
			want: consent.ErrInvalidAction,
		},
		{
			name: "empty categories after normalize",
			in: consent.NewInput{
				Categories:    []string{"", "  "},
				Action:        consent.ActionRefuseAll,
				IPAnonymized:  "1.2.x.x",
				UserAgentHash: "h",
			},
			want: consent.ErrCategoriesRequired,
		},
		{
			name: "missing ip",
			in: consent.NewInput{
				Categories:    []string{"analytics"},
				Action:        consent.ActionAcceptAll,
				IPAnonymized:  "   ",
				UserAgentHash: "h",
			},
			want: consent.ErrIPAnonymizedRequired,
		},
		{
			name: "missing UA hash",
			in: consent.NewInput{
				Categories:    []string{"analytics"},
				Action:        consent.ActionAcceptAll,
				IPAnonymized:  "1.2.x.x",
				UserAgentHash: "",
			},
			want: consent.ErrUserAgentHashRequired,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := consent.New(c.in)
			if !errors.Is(err, c.want) {
				t.Errorf("got err=%v, want %v", err, c.want)
			}
		})
	}
}

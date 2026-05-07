package receipt

import (
	"testing"

	"github.com/google/uuid"
)

func TestReceipt_IsParty(t *testing.T) {
	clientOrg := uuid.New()
	providerOrg := uuid.New()
	referrerOrg := uuid.New()
	otherOrg := uuid.New()

	cases := []struct {
		name string
		rec  *Receipt
		org  uuid.UUID
		want bool
	}{
		{
			name: "client matches",
			rec: &Receipt{
				Client:   &PartyBilling{OrganizationID: clientOrg},
				Provider: &PartyBilling{OrganizationID: providerOrg},
			},
			org:  clientOrg,
			want: true,
		},
		{
			name: "provider matches",
			rec: &Receipt{
				Client:   &PartyBilling{OrganizationID: clientOrg},
				Provider: &PartyBilling{OrganizationID: providerOrg},
			},
			org:  providerOrg,
			want: true,
		},
		{
			name: "referrer matches",
			rec: &Receipt{
				Client:   &PartyBilling{OrganizationID: clientOrg},
				Provider: &PartyBilling{OrganizationID: providerOrg},
				Referrer: &PartyBilling{OrganizationID: referrerOrg},
			},
			org:  referrerOrg,
			want: true,
		},
		{
			name: "unrelated org rejected",
			rec: &Receipt{
				Client:   &PartyBilling{OrganizationID: clientOrg},
				Provider: &PartyBilling{OrganizationID: providerOrg},
			},
			org:  otherOrg,
			want: false,
		},
		{
			name: "nil receipt rejected",
			rec:  nil,
			org:  clientOrg,
			want: false,
		},
		{
			name: "nil org rejected",
			rec: &Receipt{
				Client: &PartyBilling{OrganizationID: clientOrg},
			},
			org:  uuid.Nil,
			want: false,
		},
		{
			name: "missing snapshot rejects every check",
			rec: &Receipt{
				SnapshotAvailable: false,
			},
			org:  clientOrg,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rec.IsParty(tc.org); got != tc.want {
				t.Fatalf("IsParty(%s) = %v, want %v", tc.org, got, tc.want)
			}
		})
	}
}

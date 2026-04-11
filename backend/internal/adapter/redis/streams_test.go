package redis

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsBusyGroupErr documents the expected detection behaviour for the
// benign BUSYGROUP response Redis returns when an XGROUP CREATE targets an
// existing consumer group. The message wording has varied between Redis
// versions ("already used" on older servers, "already exists" on newer
// ones), so matching by prefix keeps the check stable across upgrades.
func TestIsBusyGroupErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error is not a busygroup",
			err:  nil,
			want: false,
		},
		{
			name: "Redis 6 wording — already used",
			err:  errors.New("BUSYGROUP Consumer Group name already used"),
			want: true,
		},
		{
			name: "Redis 7 wording — already exists",
			err:  errors.New("BUSYGROUP Consumer Group name already exists"),
			want: true,
		},
		{
			name: "future wording also detected by prefix",
			err:  errors.New("BUSYGROUP some new variant message"),
			want: true,
		},
		{
			name: "unrelated error is not a busygroup",
			err:  errors.New("NOGROUP No such key"),
			want: false,
		},
		{
			name: "connectivity error is not a busygroup",
			err:  errors.New("dial tcp: connection refused"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isBusyGroupErr(tt.err))
		})
	}
}

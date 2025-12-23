package stringsx

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClip_Table(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"short", "hello", 10, "hello"},
		{"equal", "hello", 5, "hello"},
		{"clip", "hello", 3, "hel"},
		{"zero", "hello", 0, ""},
		{"neg", "hello", -1, ""},
		{"empty", "", 3, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Clip(tt.in, tt.max))
		})
	}
}

func TestNormalize_And_IsEmpty(t *testing.T) {
	require.Equal(t, "hello", Normalize("  HeLLo  "))
	require.True(t, IsEmpty("   \n\t  "))
	require.False(t, IsEmpty(" x "))
}

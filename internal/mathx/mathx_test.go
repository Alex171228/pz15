package mathx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSum_Table(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"pos", 2, 3, 5},
		{"zero", 0, 0, 0},
		{"neg", -2, 3, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Sum(tt.a, tt.b))
		})
	}
}

func TestDivide(t *testing.T) {
	res, err := Divide(10, 2)
	require.NoError(t, err)
	require.Equal(t, 5, res)

	_, err = Divide(10, 0)
	require.Error(t, err)
}

func TestFib(t *testing.T) {
	tests := []struct {
		n   int
		exp int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{5, 5},
		{10, 55},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Fib_%d", tt.n), func(t *testing.T) {
			require.Equal(t, tt.exp, Fib(tt.n))
		})
	}
}

func TestFibFast_MatchesFib(t *testing.T) {
	for n := 0; n <= 20; n++ {
		t.Run(fmt.Sprintf("n_%d", n), func(t *testing.T) {
			require.Equal(t, Fib(n), FibFast(n))
		})
	}
}

func BenchmarkFib(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Fib(20)
	}
}

func BenchmarkFibFast(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = FibFast(20)
	}
}

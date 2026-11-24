package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithQuerySeparator(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":                          "",
		"postgres://h/db":           "postgres://h/db?",
		"postgres://h/db?":          "postgres://h/db?",
		"postgres://h/db?ssl=off":   "postgres://h/db?ssl=off&",
		"postgres://h/db?ssl=off&":  "postgres://h/db?ssl=off&",
		"postgres://h/db?a=1&b=2":   "postgres://h/db?a=1&b=2&",
		"postgres://h:5432/bms?x=1": "postgres://h:5432/bms?x=1&",
	}

	for in, want := range tests {
		assert.Equal(t, want, withQuerySeparator(in), "input %q", in)
	}
}

func TestRun_RejectsBadInput(t *testing.T) {
	t.Run("missing config", func(t *testing.T) {
		t.Setenv("CONFIG_PATH", "")
		require.ErrorContains(t, run("", nil), "config path is not set")
	})
}

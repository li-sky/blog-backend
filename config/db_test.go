package config

import "testing"

import "github.com/stretchr/testify/require"

func TestParseEnvInt_Fallback(t *testing.T) {
	t.Setenv("X", "")
	require.Equal(t, 5, parseEnvInt("X", 5))
}

func TestParseEnvInt_Invalid(t *testing.T) {
	t.Setenv("X", "nope")
	require.Equal(t, 7, parseEnvInt("X", 7))
}

func TestParseEnvInt_Value(t *testing.T) {
	t.Setenv("X", "12")
	require.Equal(t, 12, parseEnvInt("X", 0))
}

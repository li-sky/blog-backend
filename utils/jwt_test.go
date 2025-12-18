package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateParseToken_RoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	token, err := GenerateToken(42, "alice", "admin")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseToken(token)
	require.NoError(t, err)
	require.Equal(t, uint(42), claims.UserID)
	require.Equal(t, "alice", claims.Username)
	require.Equal(t, "admin", claims.Role)
	require.Equal(t, "blog-backend", claims.Issuer)
	require.NotNil(t, claims.ExpiresAt)
}

func TestParseToken_Empty(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	claims, err := ParseToken("")
	require.Error(t, err)
	require.Nil(t, claims)
}

func TestParseToken_Invalid(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	claims, err := ParseToken("not-a-jwt")
	require.Error(t, err)
	require.Nil(t, claims)
}

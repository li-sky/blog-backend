package repositories

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPostOrder_Defaults(t *testing.T) {
	order := buildPostOrder("", false)
	require.Equal(t, "created_at ASC", order)

	order = buildPostOrder("unknown", true)
	require.Equal(t, "created_at DESC", order)
}

func TestBuildPostOrder_AllowsKnownColumns(t *testing.T) {
	require.Equal(t, "title ASC", buildPostOrder("title", false))
	require.Equal(t, "published_at DESC", buildPostOrder("published_at", true))
}

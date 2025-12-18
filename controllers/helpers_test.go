package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestParseUintParam_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	r.GET("/users/:id", func(c *gin.Context) {
		id, err := parseUintParam(c, "id")
		require.NoError(t, err)
		require.Equal(t, uint(123), id)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	_ = c
}

func TestPaginationParams_Defaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?x=1", nil)

	limit, offset := paginationParams(c)
	require.Equal(t, 20, limit)
	require.Equal(t, 0, offset)
}

func TestPaginationParams_Custom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/?limit=5&offset=10", nil)

	limit, offset := paginationParams(c)
	require.Equal(t, 5, limit)
	require.Equal(t, 10, offset)
}

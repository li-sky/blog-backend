package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHealthRoutesReflectDatabaseReadiness(t *testing.T) {
	for _, path := range []string{"/health", "/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			var databaseError error
			r := gin.New()
			registerHealthRoutes(r, func(ctx context.Context) error {
				deadline, ok := ctx.Deadline()
				require.True(t, ok)
				require.InDelta(t, 2, time.Until(deadline).Seconds(), 0.5)
				return databaseError
			})
			request := func(status int, body string) {
				w := httptest.NewRecorder()
				r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
				require.Equal(t, status, w.Code)
				require.JSONEq(t, body, w.Body.String())
			}
			request(http.StatusOK, `{"status":"ok"}`)
			databaseError = errors.New("database unavailable: sensitive connection details")
			request(http.StatusServiceUnavailable, `{"status":"unavailable"}`)
			databaseError = nil
			request(http.StatusOK, `{"status":"ok"}`)
		})
	}
}

func TestHealthWithoutDatabaseCheckerIsUnavailable(t *testing.T) {
	r := gin.New()
	registerHealthRoutes(r, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHealthPropagatesRequestCancellation(t *testing.T) {
	r := gin.New()
	registerHealthRoutes(r, func(ctx context.Context) error { return ctx.Err() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil).WithContext(ctx))
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCountryByIP_Empty(t *testing.T) {
	require.Equal(t, "", CountryByIP(""))
}

func TestCountryByIP_SuccessCountryName(t *testing.T) {
	prevBase := geoipBaseURL
	prevClient := geoipHTTPClient
	t.Cleanup(func() {
		SetGeoIPBaseURLForTest(prevBase)
		SetGeoIPHTTPClientForTest(prevClient)
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"country_name":"China","country":"CN"}`))
	}))
	SetGeoIPBaseURLForTest(srv.URL)
	SetGeoIPHTTPClientForTest(srv.Client())
	defer srv.Close()

	require.Equal(t, "China", CountryByIP("1.2.3.4"))
}

func TestCountryByIP_FallbackCountryCode(t *testing.T) {
	prevBase := geoipBaseURL
	prevClient := geoipHTTPClient
	t.Cleanup(func() {
		SetGeoIPBaseURLForTest(prevBase)
		SetGeoIPHTTPClientForTest(prevClient)
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"country":"CN"}`))
	}))
	SetGeoIPBaseURLForTest(srv.URL)
	SetGeoIPHTTPClientForTest(srv.Client())
	defer srv.Close()

	require.Equal(t, "CN", CountryByIP("1.2.3.4"))
}

func TestCountryByIP_Non200(t *testing.T) {
	prevBase := geoipBaseURL
	prevClient := geoipHTTPClient
	t.Cleanup(func() {
		SetGeoIPBaseURLForTest(prevBase)
		SetGeoIPHTTPClientForTest(prevClient)
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	SetGeoIPBaseURLForTest(srv.URL)
	SetGeoIPHTTPClientForTest(srv.Client())
	defer srv.Close()

	require.Equal(t, "", CountryByIP("1.2.3.4"))
}

func TestCountryByIP_InvalidJSON(t *testing.T) {
	prevBase := geoipBaseURL
	prevClient := geoipHTTPClient
	t.Cleanup(func() {
		SetGeoIPBaseURLForTest(prevBase)
		SetGeoIPHTTPClientForTest(prevClient)
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{`))
	}))
	SetGeoIPBaseURLForTest(srv.URL)
	SetGeoIPHTTPClientForTest(srv.Client())
	defer srv.Close()

	require.Equal(t, "", CountryByIP("1.2.3.4"))
}

func TestSetGeoIPHTTPClientForTest_NilResets(t *testing.T) {
	prevClient := geoipHTTPClient
	defer SetGeoIPHTTPClientForTest(prevClient)

	SetGeoIPHTTPClientForTest(nil)
	require.NotNil(t, geoipHTTPClient)
}

package utils

import (
	"encoding/json"
	"net/http"
	"time"
)

var geoipBaseURL = "https://ipapi.co"

var geoipHTTPClient = &http.Client{Timeout: 2 * time.Second}

// SetGeoIPBaseURLForTest overrides the GeoIP base URL.
// Intended for tests to avoid real network calls.
func SetGeoIPBaseURLForTest(baseURL string) {
	geoipBaseURL = baseURL
}

// SetGeoIPHTTPClientForTest overrides the HTTP client used for GeoIP lookups.
// Intended for tests to avoid real network calls.
func SetGeoIPHTTPClientForTest(client *http.Client) {
	if client == nil {
		geoipHTTPClient = &http.Client{Timeout: 2 * time.Second}
		return
	}
	geoipHTTPClient = client
}

// CountryByIP returns country name for an IP via ipapi.co; returns empty string on failure.
func CountryByIP(ip string) string {
	if ip == "" {
		return ""
	}
	url := geoipBaseURL + "/" + ip + "/json/"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	resp, err := geoipHTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var parsed struct {
		CountryName string `json:"country_name"`
		Country     string `json:"country"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ""
	}
	if parsed.CountryName != "" {
		return parsed.CountryName
	}
	return parsed.Country
}

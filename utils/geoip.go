package utils

import (
	"encoding/json"
	"net/http"
	"time"
)

// CountryByIP returns country name for an IP via ipapi.co; returns empty string on failure.
func CountryByIP(ip string) string {
	if ip == "" {
		return ""
	}
	client := &http.Client{Timeout: 2 * time.Second}
	url := "https://ipapi.co/" + ip + "/json/"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	var parsed struct {
		CountryName string `json:"country_name"`
		Country    string `json:"country"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return ""
	}
	if parsed.CountryName != "" {
		return parsed.CountryName
	}
	return parsed.Country
}

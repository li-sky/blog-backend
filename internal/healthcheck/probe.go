// Package healthcheck supplies an HTTP probe for the shell-free runtime image.
package healthcheck

import (
	"context"
	"fmt"
	"net/http"
)

// Probe performs one GET and requires an HTTP 200 response, without a proxy or redirects.
func Probe(ctx context.Context, endpoint string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create health probe: %w", err)
	}
	client := &http.Client{
		Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("health probe failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health probe returned HTTP %d", response.StatusCode)
	}
	return nil
}

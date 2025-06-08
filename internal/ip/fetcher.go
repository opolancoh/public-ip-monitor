package ip

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Fetcher handles fetching current public IP from external services
type Fetcher struct {
	services   []string
	timeout    time.Duration
	httpClient *http.Client
}

// NewFetcher creates a new IP fetcher
func NewFetcher(services []string, timeoutSeconds int) *Fetcher {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Fetcher{
		services: services,
		timeout:  timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetCurrentIP fetches the current public IP from external services
func (f *Fetcher) GetCurrentIP(ctx context.Context) (string, error) {
	if len(f.services) == 0 {
		return "", fmt.Errorf("no IP services configured")
	}

	// Try multiple services for reliability
	var lastError error
	for _, service := range f.services {
		ip, err := f.fetchFromService(ctx, service)
		if err != nil {
			lastError = err
			continue
		}
		return ip, nil
	}

	return "", fmt.Errorf("failed to get IP from all services, last error: %w", lastError)
}

// fetchFromService fetches IP from a specific service
func (f *Fetcher) fetchFromService(ctx context.Context, serviceURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", serviceURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %w", serviceURL, err)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch from %s: %w", serviceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("service %s returned status %d", serviceURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response from %s: %w", serviceURL, err)
	}

	// Clean up response (remove newlines, whitespace, etc.)
	ip := strings.TrimSpace(string(body))

	// Basic validation
	if ip == "" {
		return "", fmt.Errorf("empty response from %s", serviceURL)
	}

	return ip, nil
}

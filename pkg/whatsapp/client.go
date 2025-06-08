package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MetaClient implements WhatsApp client using Meta Business API
type MetaClient struct {
	config     Config
	httpClient *http.Client
}

// MetaFactory creates Meta WhatsApp clients
type MetaFactory struct{}

// NewMetaFactory creates a new Meta factory
func NewMetaFactory() *MetaFactory {
	return &MetaFactory{}
}

// NewClient creates a new Meta WhatsApp client
func (f *MetaFactory) NewClient(config Config) (Client, error) {
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &MetaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Send sends a WhatsApp message using Meta Business API
func (c *MetaClient) Send(ctx context.Context, message Message) error {
	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages",
		c.config.APIVersion, c.config.PhoneID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                message.To,
		"type":              "text",
		"text": map[string]string{
			"body": message.Text,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// Close closes the WhatsApp client
func (c *MetaClient) Close() error {
	return nil
}

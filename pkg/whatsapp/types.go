package whatsapp

import "context"

// Message represents a WhatsApp message
type Message struct {
	To   string
	Text string
}

// Config represents WhatsApp configuration
type Config struct {
	Token          string
	PhoneID        string
	APIVersion     string
	TimeoutSeconds int
}

// Client defines the WhatsApp client interface
type Client interface {
	Send(ctx context.Context, message Message) error
	Close() error
}

// Factory creates WhatsApp clients
type Factory interface {
	NewClient(config Config) (Client, error)
}

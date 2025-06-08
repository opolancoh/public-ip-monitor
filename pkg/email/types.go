package email

import "context"

// Message represents an email message
type Message struct {
	To      string
	Subject string
	Body    string
}

// Config represents email configuration
type Config struct {
	From     string
	Password string
	SMTPHost string
	SMTPPort string
	Timeout  int
}

// Client defines the email client interface
type Client interface {
	Send(ctx context.Context, message Message) error
	Close() error
}

// Factory creates email clients
type Factory interface {
	NewClient(config Config) (Client, error)
}

package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"time"
)

// SMTPClient implements the email client using SMTP
type SMTPClient struct {
	config Config
}

// SMTPFactory creates SMTP email clients
type SMTPFactory struct{}

// NewSMTPFactory creates a new SMTP factory
func NewSMTPFactory() *SMTPFactory {
	return &SMTPFactory{}
}

// NewClient creates a new SMTP email client
func (f *SMTPFactory) NewClient(config Config) (Client, error) {
	return &SMTPClient{
		config: config,
	}, nil
}

// Send sends an email using SMTP
func (c *SMTPClient) Send(ctx context.Context, message Message) error {
	// Create context with timeout
	if c.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(c.config.Timeout)*time.Second)
		defer cancel()
	}

	// Set up authentication
	auth := smtp.PlainAuth("", c.config.From, c.config.Password, c.config.SMTPHost)

	// Prepare email message
	msg := []byte(fmt.Sprintf(
		"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		message.To, message.Subject, message.Body))

	// SMTP server address
	addr := c.config.SMTPHost + ":" + c.config.SMTPPort

	// Connect to SMTP server
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Quit()

	// Start TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         c.config.SMTPHost,
	}

	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender
	if err = conn.Mail(c.config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipient
	if err = conn.Rcpt(message.To); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	// Send the email body
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to send email data: %w", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close email writer: %w", err)
	}

	return nil
}

// Close closes the email client (no-op for SMTP)
func (c *SMTPClient) Close() error {
	return nil
}

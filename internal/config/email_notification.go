package config

import (
	"fmt"
	"time"
)

// BuildEmailSubject creates the email subject line
func BuildEmailSubject() string {
	return "🚨 IP Address Changed - Public IP Monitor"
}

// BuildEmailBody creates the email body content
func BuildEmailBody(oldIP, newIP string, timestamp time.Time) string {
	return fmt.Sprintf(`IP Address Change Notification

Your public IP address has changed:

Previous IP: %s
New IP: %s
Change Time: %s

This notification was sent automatically by your IP monitoring service.

Best regards,
Public IP Monitor`, oldIP, newIP, timestamp.Format("2006-01-02 15:04:05"))
}

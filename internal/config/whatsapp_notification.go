package config

import (
	"fmt"
	"time"
)

// BuildWhatsAppMessage creates the WhatsApp message content
func BuildWhatsAppMessage(oldIP, newIP string, timestamp time.Time) string {
	return fmt.Sprintf("🚨 IP Address Changed!\n\nOld IP: %s\nNew IP: %s\nTime: %s\n\nPublic IP Monitor",
		oldIP, newIP, timestamp.Format("2006-01-02 15:04:05"))
}

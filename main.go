// Package main implements a dynamic IP monitoring service that tracks changes
// to the public IP address and sends notifications via WhatsApp and email.
//
// The service polls multiple IP detection services for redundancy and maintains
// a historical record of IP changes with timestamps. Configuration is managed
// through a JSON file with sensible defaults.
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

const (
	// File paths for persistent storage
	ConfigFile  = "config.json"
	RecordsFile = "ip_records.json"
	LastIpFile  = "last_ip.txt"

	// Default monitoring interval
	DefaultCheckIntervalSeconds = 300

	// WhatsApp Business API defaults
	DefaultWhatsappToken   = "YOUR_WHATSAPP_TOKEN"
	DefaultWhatsappPhoneId = "YOUR_PHONE_ID"
	DefaultRecipientNumber = "YOUR_RECIPIENT_NUMBER"

	// SMTP configuration defaults
	DefaultEmailFrom     = "your-email@gmail.com"
	DefaultEmailPassword = "your-app-password"
	DefaultEmailTo       = "recipient@gmail.com"
	DefaultSMTPHost      = "smtp.gmail.com"
	DefaultSMTPPort      = "587"

	// Network and API configuration
	HttpTimeoutSeconds = 30
	WhatsappApiVersion = "v17.0"

	// File system permissions
	ConfigFilePerm = 0644
	DataFilePerm   = 0644
)

// IP_SERVICES provides multiple endpoints for IP detection to ensure reliability
// in case one service becomes unavailable.
var IP_SERVICES = []string{
	"https://api.ipify.org",
	"https://icanhazip.com",
	"https://ipecho.net/plain",
}

// IPRecord represents a single IP change event with timestamp for historical tracking.
type IPRecord struct {
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

// Config holds all application configuration including notification settings
// and monitoring parameters. Both WhatsApp and email notifications can be
// independently enabled or disabled.
type Config struct {
	CheckIntervalSeconds int `json:"check_interval_seconds"`

	// WhatsApp Business API configuration
	WhatsAppEnabled bool   `json:"whatsapp_enabled"`
	WhatsAppToken   string `json:"whatsapp_token"`
	WhatsAppPhoneID string `json:"whatsapp_phone_id"`
	RecipientNumber string `json:"recipient_number"`

	// SMTP email configuration
	EmailEnabled  bool   `json:"email_enabled"`
	EmailFrom     string `json:"email_from"`
	EmailPassword string `json:"email_password"`
	EmailTo       string `json:"email_to"`
	SMTPHost      string `json:"smtp_host"`
	SMTPPort      string `json:"smtp_port"`
}

func main() {
	log.Println("Starting IP Monitor...")

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	lastIP := readLastIP()
	if lastIP == "" {
		log.Println("No last IP found - this appears to be the first run")
	} else {
		log.Printf("Last known IP: %s", lastIP)
	}

	// Perform initial check to establish baseline
	checkIPAndNotify(config)

	// Set up continuous monitoring
	ticker := time.NewTicker(time.Duration(config.CheckIntervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Printf("Monitoring IP changes every %d seconds...", config.CheckIntervalSeconds)

	for range ticker.C {
		checkIPAndNotify(config)
	}
}

// loadConfig reads configuration from JSON file, creating a default configuration
// file with placeholder values if none exists. Returns an error prompting user
// to fill in credentials when default config is created.
func loadConfig() (*Config, error) {
	if _, err := os.Stat(ConfigFile); os.IsNotExist(err) {
		defaultConfig := &Config{
			WhatsAppToken:        DefaultWhatsappToken,
			WhatsAppPhoneID:      DefaultWhatsappPhoneId,
			RecipientNumber:      DefaultRecipientNumber,
			CheckIntervalSeconds: DefaultCheckIntervalSeconds,
			WhatsAppEnabled:      false,
			EmailEnabled:         true,
			EmailFrom:            DefaultEmailFrom,
			EmailPassword:        DefaultEmailPassword,
			EmailTo:              DefaultEmailTo,
			SMTPHost:             DefaultSMTPHost,
			SMTPPort:             DefaultSMTPPort,
		}

		data, _ := json.MarshalIndent(defaultConfig, "", "    ")
		os.WriteFile(ConfigFile, data, ConfigFilePerm)

		return nil, fmt.Errorf("created default config.json - please fill in your WhatsApp and email credentials")
	}

	data, err := os.ReadFile(ConfigFile)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	return &config, err
}

// getCurrentIP attempts to retrieve the public IP address from multiple services
// for redundancy. Returns the first successful response or an error if all services fail.
func getCurrentIP() (string, error) {
	for _, service := range IP_SERVICES {
		resp, err := http.Get(service)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}

			ip := strings.TrimSpace(string(body))
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get IP from all services")
}

// readLastIP retrieves the previously stored IP address from persistent storage.
// Returns empty string if no previous IP exists or file cannot be read.
func readLastIP() string {
	data, err := os.ReadFile(LastIpFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveLastIP persists the current IP address to disk for comparison on next run.
func saveLastIP(ip string) error {
	return os.WriteFile(LastIpFile, []byte(ip), DataFilePerm)
}

// saveIPRecord appends a new IP change record to the historical log.
// Maintains a JSON array of all IP changes with timestamps.
func saveIPRecord(ip string) error {
	record := IPRecord{
		IP:        ip,
		Timestamp: time.Now(),
	}

	var records []IPRecord
	if data, err := os.ReadFile(RecordsFile); err == nil {
		json.Unmarshal(data, &records)
	}

	records = append(records, record)

	data, err := json.MarshalIndent(records, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(RecordsFile, data, DataFilePerm)
}

// sendWhatsAppMessage delivers notifications via WhatsApp Business API.
// Requires valid API token and phone number configuration. Skips silently
// if WhatsApp notifications are disabled.
func sendWhatsAppMessage(config *Config, message string) error {
	if !config.WhatsAppEnabled {
		return nil
	}

	url := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", WhatsappApiVersion, config.WhatsAppPhoneID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                config.RecipientNumber,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+config.WhatsAppToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: HttpTimeoutSeconds * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WhatsApp API error: %s", string(body))
	}

	return nil
}

// sendEmail delivers notifications via SMTP with STARTTLS encryption.
// Handles the complete SMTP conversation including TLS negotiation and authentication.
// Skips silently if email notifications are disabled.
func sendEmail(config *Config, subject, body string) error {
	if !config.EmailEnabled {
		return nil
	}

	auth := smtp.PlainAuth("", config.EmailFrom, config.EmailPassword, config.SMTPHost)

	msg := []byte(fmt.Sprintf(
		"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		config.EmailTo, subject, body))

	addr := config.SMTPHost + ":" + config.SMTPPort

	// Establish plain connection then upgrade to TLS
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer conn.Quit()

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         config.SMTPHost,
	}

	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %v", err)
	}

	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %v", err)
	}

	if err = conn.Mail(config.EmailFrom); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	if err = conn.Rcpt(config.EmailTo); err != nil {
		return fmt.Errorf("failed to set recipient: %v", err)
	}

	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to send email data: %v", err)
	}

	_, err = w.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to write email message: %v", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close email writer: %v", err)
	}

	return nil
}

// checkIPAndNotify performs the core monitoring logic by comparing current IP
// against the last known IP and triggering notifications on changes.
func checkIPAndNotify(config *Config) {
	currentIP, err := getCurrentIP()
	if err != nil {
		log.Printf("Error getting current IP: %v", err)
		return
	}

	log.Printf("Current IP: %s", currentIP)

	lastIP := readLastIP()

	if currentIP != lastIP {
		displayLastIP := lastIP
		if displayLastIP == "" {
			displayLastIP = "Unknown"
		}

		log.Printf("IP changed from %s to %s", displayLastIP, currentIP)
		notifyIPChange(config, lastIP, currentIP)
	} else {
		log.Println("IP unchanged")
	}
}

// notifyIPChange coordinates the notification process when an IP change is detected.
// Updates persistent storage and dispatches notifications through all enabled channels.
func notifyIPChange(config *Config, lastIP, currentIP string) {
	if err := saveLastIP(currentIP); err != nil {
		log.Printf("Error saving last IP: %v", err)
	}

	if err := saveIPRecord(currentIP); err != nil {
		log.Printf("Error saving IP record: %v", err)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Dispatch WhatsApp notification
	if config.WhatsAppEnabled {
		whatsappMessage := fmt.Sprintf("🚨 IP Address Changed!\n\nOld IP: %s\nNew IP: %s\nTime: %s\n\nRaspberry Pi Monitor",
			lastIP, currentIP, timestamp)

		if err := sendWhatsAppMessage(config, whatsappMessage); err != nil {
			log.Printf("Error sending WhatsApp message: %v", err)
		} else {
			log.Println("WhatsApp notification sent successfully")
		}
	}

	// Dispatch email notification
	if config.EmailEnabled {
		emailSubject := "🚨 IP Address Changed - Raspberry Pi Monitor"
		emailBody := fmt.Sprintf(`IP Address Change Notification

Your Raspberry Pi's public IP address has changed:

Previous IP: %s
New IP: %s
Change Time: %s

This notification was sent automatically by your IP monitoring service.

Best regards,
Raspberry Pi Monitor`, lastIP, currentIP, timestamp)

		if err := sendEmail(config, emailSubject, emailBody); err != nil {
			log.Printf("Error sending email notification: %v", err)
		} else {
			log.Println("Email notification sent successfully")
		}
	}
}

// getIPHistory retrieves the complete historical record of IP changes
// from persistent storage. Returns empty slice if no history exists.
func getIPHistory() ([]IPRecord, error) {
	var records []IPRecord

	data, err := os.ReadFile(RecordsFile)
	if err != nil {
		return records, err
	}

	err = json.Unmarshal(data, &records)
	return records, err
}

// printIPHistory outputs the complete IP change history to stdout.
// Useful for debugging and manual inspection of change patterns.
func printIPHistory() {
	records, err := getIPHistory()
	if err != nil {
		log.Printf("Error reading IP history: %v", err)
		return
	}

	fmt.Println("\n=== IP Change History ===")
	for i, record := range records {
		fmt.Printf("%d. IP: %s - Time: %s\n",
			i+1, record.IP, record.Timestamp.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("========================")
}

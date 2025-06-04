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

// =====================================
// CONFIGURATION AND CONSTANTS
// =====================================

const (
	// File paths
	ConfigFile  = "config.json"
	RecordsFile = "ip_records.json"
	LastIpFile  = "last_ip.txt"

	// Default configuration values
	DefaultCheckIntervalSeconds = 300 // 5 minutes in seconds
	DefaultWhatsappToken        = "YOUR_WHATSAPP_TOKEN"
	DefaultWhatsappPhoneId      = "YOUR_PHONE_ID"
	DefaultRecipientNumber      = "YOUR_RECIPIENT_NUMBER"

	// Email defaults
	DefaultEmailFrom     = "your-email@gmail.com"
	DefaultEmailPassword = "your-app-password"
	DefaultEmailTo       = "recipient@gmail.com"
	DefaultSMTPHost      = "smtp.gmail.com"
	DefaultSMTPPort      = "587"

	// API settings
	HttpTimeoutSeconds = 30
	WhatsappApiVersion = "v17.0"

	// File permissions
	ConfigFilePerm = 0644
	DataFilePerm   = 0644
)

// IP services for redundancy
var IP_SERVICES = []string{
	"https://api.ipify.org",
	"https://icanhazip.com",
	"https://ipecho.net/plain",
}

// =====================================
// DATA STRUCTURES
// =====================================

// IPRecord represents an IP change record
type IPRecord struct {
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

// Config holds configuration for the application
type Config struct {
	CheckIntervalSeconds int `json:"check_interval_seconds"`

	// WhatsApp configuration
	WhatsAppEnabled bool   `json:"whatsapp_enabled"`
	WhatsAppToken   string `json:"whatsapp_token"`
	WhatsAppPhoneID string `json:"whatsapp_phone_id"`
	RecipientNumber string `json:"recipient_number"`

	// Email configuration
	EmailEnabled  bool   `json:"email_enabled"`
	EmailFrom     string `json:"email_from"`
	EmailPassword string `json:"email_password"`
	EmailTo       string `json:"email_to"`
	SMTPHost      string `json:"smtp_host"`
	SMTPPort      string `json:"smtp_port"`
}

func main() {
	log.Println("Starting IP Monitor...")

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Read last known IP
	lastIP := readLastIP()
	if lastIP == "" {
		log.Println("No last IP found - this appears to be the first run")
	} else {
		log.Printf("Last known IP: %s", lastIP)
	}

	// Check IP immediately on startup
	checkIPAndNotify(config)

	// Set up periodic checking
	ticker := time.NewTicker(time.Duration(config.CheckIntervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Printf("Monitoring IP changes every %d seconds...", config.CheckIntervalSeconds)

	for range ticker.C {
		checkIPAndNotify(config)
	}
}

// loadConfig loads configuration from JSON file
func loadConfig() (*Config, error) {
	// Create default config file if it doesn't exist
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

// getCurrentIP fetches the current public IP
func getCurrentIP() (string, error) {
	// Try multiple services for reliability
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

			ip := string(body)
			// Clean up response (remove newlines, etc.)
			ip = strings.TrimSpace(ip)
			return ip, nil
		}
	}

	return "", fmt.Errorf("failed to get IP from all services")
}

// readLastIP reads the last known IP from file
func readLastIP() string {
	data, err := os.ReadFile(LastIpFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// saveLastIP saves the current IP to file
func saveLastIP(ip string) error {
	return os.WriteFile(LastIpFile, []byte(ip), DataFilePerm)
}

// saveIPRecord adds a new IP change record
func saveIPRecord(ip string) error {
	record := IPRecord{
		IP:        ip,
		Timestamp: time.Now(),
	}

	// Read existing records
	var records []IPRecord
	if data, err := os.ReadFile(RecordsFile); err == nil {
		json.Unmarshal(data, &records)
	}

	// Add new record
	records = append(records, record)

	// Save updated records
	data, err := json.MarshalIndent(records, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(RecordsFile, data, DataFilePerm)
}

// sendWhatsAppMessage sends a WhatsApp message using Meta's Business API
func sendWhatsAppMessage(config *Config, message string) error {
	if !config.WhatsAppEnabled {
		return nil // WhatsApp is disabled
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

// sendEmail sends an email notification using SMTP
func sendEmail(config *Config, subject, body string) error {
	if !config.EmailEnabled {
		return nil // Email is disabled
	}

	// Set up authentication information
	auth := smtp.PlainAuth("", config.EmailFrom, config.EmailPassword, config.SMTPHost)

	// Email headers and body
	msg := []byte(fmt.Sprintf(
		"To: %s\r\n"+
			"Subject: %s\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		config.EmailTo, subject, body))

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step
	addr := config.SMTPHost + ":" + config.SMTPPort

	// Connect to SMTP server with plain connection first
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer conn.Quit()

	// Start TLS (STARTTLS)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         config.SMTPHost,
	}

	if err = conn.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %v", err)
	}

	// Authenticate
	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %v", err)
	}

	// Set sender
	if err = conn.Mail(config.EmailFrom); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set recipient
	if err = conn.Rcpt(config.EmailTo); err != nil {
		return fmt.Errorf("failed to set recipient: %v", err)
	}

	// Send the email body
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

// checkIPAndNotify checks current IP and triggers notification if changed
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

// notifyIPChange handles the notification process when IP changes
func notifyIPChange(config *Config, lastIP, currentIP string) {
	// Save new IP
	if err := saveLastIP(currentIP); err != nil {
		log.Printf("Error saving last IP: %v", err)
	}

	// Save record
	if err := saveIPRecord(currentIP); err != nil {
		log.Printf("Error saving IP record: %v", err)
	}

	// Prepare notification message
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Send WhatsApp notification
	if config.WhatsAppEnabled {
		whatsappMessage := fmt.Sprintf("🚨 IP Address Changed!\n\nOld IP: %s\nNew IP: %s\nTime: %s\n\nRaspberry Pi Monitor",
			lastIP, currentIP, timestamp)

		if err := sendWhatsAppMessage(config, whatsappMessage); err != nil {
			log.Printf("Error sending WhatsApp message: %v", err)
		} else {
			log.Println("WhatsApp notification sent successfully")
		}
	}

	// Send Email notification
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

// Additional utility functions you might want

// getIPHistory returns the history of IP changes
func getIPHistory() ([]IPRecord, error) {
	var records []IPRecord

	data, err := os.ReadFile(RecordsFile)
	if err != nil {
		return records, err
	}

	err = json.Unmarshal(data, &records)
	return records, err
}

// printIPHistory prints the IP change history (useful for debugging)
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

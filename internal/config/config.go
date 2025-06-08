package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultConfigFile = "config.json"
	ConfigFilePerm    = 0644
)

// Manager handles configuration loading and saving
type Manager struct {
	configPath string
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	if configPath == "" {
		configPath = DefaultConfigFile
	}
	return &Manager{
		configPath: configPath,
	}
}

// Load loads configuration from file
func (m *Manager) Load() (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Create default config
		defaultConfig := m.createDefaultConfig()
		if err := m.Save(defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return nil, fmt.Errorf("created default config file at %s - please update with your credentials", m.configPath)
	}

	// Read existing config
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate and set defaults
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save saves configuration to file
func (m *Manager) Save(config *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, ConfigFilePerm); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCheckInterval returns the check interval as a duration
func GetCheckInterval(config *Config) time.Duration {
	return time.Duration(config.CheckIntervalSeconds) * time.Second
}

// validateConfig validates the configuration and sets defaults
func validateConfig(c *Config) error {
	if c.CheckIntervalSeconds <= 0 {
		c.CheckIntervalSeconds = 300 // Default 5 minutes
	}

	if c.Logging.Timezone == "" {
		c.Logging.Timezone = "UTC"
	}

	if c.Logging.Format == "" {
		c.Logging.Format = "2006-01-02 15:04:05"
	}

	if c.WhatsApp.APIVersion == "" {
		c.WhatsApp.APIVersion = "v17.0"
	}

	if c.WhatsApp.TimeoutSeconds <= 0 {
		c.WhatsApp.TimeoutSeconds = 30
	}

	if c.Email.SMTPPort == "" {
		c.Email.SMTPPort = "587"
	}

	if c.Email.Timeout <= 0 {
		c.Email.Timeout = 30
	}

	if c.IP.TimeoutSeconds <= 0 {
		c.IP.TimeoutSeconds = 30
	}

	if c.IP.DataDir == "" {
		c.IP.DataDir = "data"
	}

	if c.IP.RecordsFile == "" {
		c.IP.RecordsFile = "ip_records.json"
	}

	if c.IP.LastIPFile == "" {
		c.IP.LastIPFile = "last_ip.txt"
	}

	if len(c.IP.Services) == 0 {
		c.IP.Services = []string{
			"https://api.ipify.org",
			"https://icanhazip.com",
			"https://ipecho.net/plain",
		}
	}

	return nil
}

// createDefaultConfig creates a default configuration
func (m *Manager) createDefaultConfig() *Config {
	return &Config{
		CheckIntervalSeconds: 300, // 5 minutes
		Logging: LoggingConfig{
			Timezone: "UTC",
			Format:   "2006-01-02 15:04:05",
		},
		WhatsApp: WhatsAppConfig{
			Enabled:         false,
			Token:           "YOUR_WHATSAPP_TOKEN",
			PhoneID:         "YOUR_PHONE_ID",
			RecipientNumber: "YOUR_RECIPIENT_NUMBER",
			APIVersion:      "v17.0",
			TimeoutSeconds:  30,
		},
		Email: EmailConfig{
			Enabled:  true,
			From:     "your-email@gmail.com",
			Password: "your-app-password",
			To:       "recipient@gmail.com",
			SMTPHost: "smtp.gmail.com",
			SMTPPort: "587",
			Timeout:  30,
		},
		IP: IPConfig{
			Services: []string{
				"https://api.ipify.org",
				"https://icanhazip.com",
				"https://ipecho.net/plain",
			},
			TimeoutSeconds: 30,
			DataDir:        "data",
			RecordsFile:    "ip_records.json",
			LastIPFile:     "last_ip.txt",
		},
	}
}

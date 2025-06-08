package config

// Config holds configuration for the application
type Config struct {
	CheckIntervalSeconds int `json:"check_interval_seconds"`

	// Logging configuration
	Logging LoggingConfig `json:"logging"`

	// WhatsApp configuration
	WhatsApp WhatsAppConfig `json:"whatsapp"`

	// Email configuration
	Email EmailConfig `json:"email"`

	// IP monitoring configuration
	IP IPConfig `json:"ip"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Timezone string `json:"timezone"` // e.g., "America/New_York", "UTC"
	Format   string `json:"format"`   // e.g., "2006-01-02 15:04:05"
}

// WhatsAppConfig holds WhatsApp configuration
type WhatsAppConfig struct {
	Enabled         bool   `json:"enabled"`
	Token           string `json:"token"`
	PhoneID         string `json:"phone_id"`
	RecipientNumber string `json:"recipient_number"`
	APIVersion      string `json:"api_version"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

// EmailConfig holds email configuration
type EmailConfig struct {
	Enabled  bool   `json:"enabled"`
	From     string `json:"from"`
	Password string `json:"password"`
	To       string `json:"to"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort string `json:"smtp_port"`
	Timeout  int    `json:"timeout_seconds"`
}

// IPConfig holds IP monitoring configuration
type IPConfig struct {
	Services       []string `json:"services"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	DataDir        string   `json:"data_dir"`
	RecordsFile    string   `json:"records_file"`
	LastIPFile     string   `json:"last_ip_file"`
}

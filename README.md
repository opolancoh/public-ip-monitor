# Public IP Monitor

A Go application that continuously monitors your public IP address and sends instant notifications when changes are detected. Built with clean architecture principles and fully independent, reusable components for maximum reliability and maintainability.

## Features

- **Continuous IP Monitoring** - Monitors your public IP using multiple services for enhanced reliability and fault tolerance
- **Email Notifications** - SMTP email alerts with customizable HTML/text messages and error handling
- **WhatsApp Notifications** - Meta Business API integration for instant messaging with delivery confirmation
- **Timezone-Aware Logging** - Custom logger with configurable timezone support and structured output
- **IP Change History** - Persistent storage and comprehensive history tracking with timestamps
- **Flexible Configuration** - JSON-based configuration with validation and environment variable support
- **Graceful Shutdown** - Proper signal handling (SIGTERM/SIGINT) and resource cleanup
- **Modular Design** - Independent, reusable packages following Go best practices
- **Error Resilience** - Retry mechanisms and fallback strategies for network failures
- **Performance Optimized** - Efficient polling with configurable intervals and minimal resource usage

## 📋 Prerequisites

- **Go 1.19+** - Required for building the application
- **Network Access** - For querying public IP services
- **SMTP Credentials** - For email notifications (optional)
- **Meta Business API Token** - For WhatsApp notifications (optional)

## 🚀 Quick Start

### 1. Clone and Build

```bash
# Clone the repository
git clone <repository-url>
cd public-ip-monitor

# Build the application into a standalone executable
go build -o bin/public-ip-monitor cmd/main.go

# Alternative: Build for specific platform (example for Raspberry Pi)
GOOS=linux GOARCH=arm GOARM=7 go build -o bin/public-ip-monitor cmd/main.go
```

### 2. Initial Setup

Run the application for the first time to generate the default configuration:

```bash
./bin/public-ip-monitor
```

This creates a `config.json` file with default settings. The application will exit after creating the config file, prompting you to customize it.

### 3. Configure Your Settings

Edit the generated `config.json` file with your specific settings:

```json
{
    "check_interval_seconds": 300,
    "logging": {
        "timezone": "UTC",
        "format": "2006-01-02 15:04:05",
        "identifier": "PUBLIC-IP-MONITOR"
    },
    "email": {
        "enabled": true,
        "from": "your-email@gmail.com",
        "password": "your-app-password", 
        "to": "recipient@gmail.com",
        "smtp_host": "smtp.gmail.com",
        "smtp_port": "587",
        "timeout": 30
    },
    "whatsapp": {
        "enabled": false,
        "token": "YOUR_WHATSAPP_TOKEN",
        "phone_id": "YOUR_PHONE_ID",
        "recipient_number": "YOUR_RECIPIENT_NUMBER",
        "api_version": "v17.0",
        "timeout_seconds": 30
    },
    "ip": {
        "services": [
            "https://api.ipify.org",
            "https://icanhazip.com", 
            "https://ipecho.net/plain"
        ],
        "timeout_seconds": 30,
        "data_dir": "data",
        "records_file": "ip_records.json",
        "last_ip_file": "last_ip.txt"
    }
}
```

#### Configuration Options

| Field | Description | Default | Required |
|-------|-------------|---------|----------|
| `check_interval_seconds` | How often to check IP (in seconds) | 300 | Yes |
| `logging.timezone` | Timezone for log timestamps | "UTC" | No |
| `logging.format` | Go time format for logs | "2006-01-02 15:04:05" | No |
| `logging.identifier` | Log identifier prefix | "PUBLIC-IP-MONITOR" | No |
| `email.enabled` | Enable email notifications | true | No |
| `email.from` | Sender email address | "your-email@gmail.com" | If email enabled |
| `email.password` | App password (not regular password) | "your-app-password" | If email enabled |
| `email.to` | Recipient email address | "recipient@gmail.com" | If email enabled |
| `email.smtp_host` | SMTP server hostname | "smtp.gmail.com" | If email enabled |
| `email.smtp_port` | SMTP server port | "587" | If email enabled |
| `email.timeout` | SMTP timeout in seconds | 30 | No |
| `whatsapp.enabled` | Enable WhatsApp notifications | false | No |
| `whatsapp.token` | WhatsApp Business API token | "YOUR_WHATSAPP_TOKEN" | If WhatsApp enabled |
| `whatsapp.phone_id` | Phone number ID from Meta | "YOUR_PHONE_ID" | If WhatsApp enabled |
| `whatsapp.recipient_number` | Recipient's WhatsApp number | "YOUR_RECIPIENT_NUMBER" | If WhatsApp enabled |
| `whatsapp.api_version` | WhatsApp API version | "v17.0" | No |
| `whatsapp.timeout_seconds` | WhatsApp API timeout in seconds | 30 | No |
| `ip.services` | List of IP detection services | Multiple services | No |
| `ip.timeout_seconds` | Timeout for IP service requests | 30 | No |
| `ip.data_dir` | Directory for storing data files | "data" | No |
| `ip.records_file` | Filename for IP change records | "ip_records.json" | No |
| `ip.last_ip_file` | Filename for last known IP | "last_ip.txt" | No |

### 4. Setup Email Notifications (Optional)

For Gmail users:
1. Enable 2-factor authentication on your Google account
2. Generate an App Password: Google Account Settings → Security → 2-Step Verification → App Passwords
3. Use the generated app password in the `email.password` field

For other email providers, update the SMTP settings accordingly.

### 5. Setup WhatsApp Notifications (Optional)

1. Create a Meta Business account
2. Set up WhatsApp Business API
3. Obtain your access token and phone number ID
4. Add the recipient's phone number (include country code, no + sign)

### 6. Start Monitoring

Run the application to begin continuous monitoring:

```bash
./bin/public-ip-monitor
```

The application will:
- Check your current public IP address
- Store it for comparison
- Monitor for changes at the configured interval
- Send notifications when changes are detected
- Log all activities with timestamps

## 🏗️ Architecture

The application follows clean architecture principles with clear separation of concerns:

```
public-ip-monitor/
├── cmd/                    # Application entry point and CLI handling
│   └── main.go            # Main application logic and argument parsing
├── internal/               # Private application code (not importable)
│   ├── config/            # Configuration management and validation
│   │   ├── config.go      # Configuration struct and loading logic
│   │   └── validation.go  # Configuration validation rules
│   ├── ip/                # IP monitoring core logic
│   │   ├── monitor.go     # Main monitoring loop and state management
│   │   ├── fetcher.go     # Public IP fetching from multiple sources
│   │   └── history.go     # IP change history persistence
│   └── logger/            # Custom logging with timezone support
│       ├── logger.go      # Logger implementation
│       └── formatter.go   # Custom log formatting
└── pkg/                   # Reusable packages (importable by other projects)
    ├── email/             # Email client (fully independent)
    │   ├── client.go      # SMTP email client implementation
    │   └── templates.go   # Email template management
    └── whatsapp/          # WhatsApp client (fully independent)
        ├── client.go      # Meta Business API client
        └── messages.go    # Message formatting and sending
```

## 📖 Usage

### Local Development

```bash
# Run directly with Go
go run cmd/main.go
```

### Command Line Options

```bash
# Standard continuous monitoring
./bin/public-ip-monitor

# Check IP address once and exit (useful for testing)
./bin/public-ip-monitor -check

# Display IP change history
./bin/public-ip-monitor -history

# Use custom configuration file
./bin/public-ip-monitor -config=/path/to/your/config.json

# Display help information
./bin/public-ip-monitor -help

# Display version information
./bin/public-ip-monitor -version
```

### Example Output

```
[PUBLIC-IP-MONITOR] 2025-06-08 15:30:00 [INFO] Public IP Monitor starting...
[PUBLIC-IP-MONITOR] 2025-06-08 15:30:00 [INFO] Current IP: 203.0.113.45
[PUBLIC-IP-MONITOR] 2025-06-08 15:30:00 [INFO] Monitoring every 300 seconds...
[PUBLIC-IP-MONITOR] 2025-06-08 15:35:15 [INFO] IP changed: 203.0.113.45 → 198.51.100.123
[PUBLIC-IP-MONITOR] 2025-06-08 15:35:15 [INFO] Email notification sent successfully
[PUBLIC-IP-MONITOR] 2025-06-08 15:35:16 [INFO] WhatsApp notification sent successfully
```

## 🚀 Deployment

# Build and run

```bash
go build -ldflags "-X main.version=1.0.0" -o bin/public-ip-monitor cmd/main.go
./bin/public-ip-monitor
```

### Cross-Platform Compilation

```bash
# Linux (x64)
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=1.0.0" -o bin/public-ip-monitor-linux-amd64 cmd/main.go

# Linux (ARM - Raspberry Pi)
GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-X main.version=1.0.0" -o bin/public-ip-monitor-linux-arm7 cmd/main.go

# Windows (x64)
GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=1.0.0" -o bin/public-ip-monitor-windows-amd64.exe cmd/main.go

# macOS (x64)
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=1.0.0" -o bin/public-ip-monitor-darwin-amd64 cmd/main.go

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=1.0.0" -o bin/public-ip-monitor-darwin-arm64 cmd/main.go
```

### Server Deployment

#### 1. Prepare the Server

```bash
# Create application directory with proper permissions
sudo mkdir -p /opt/public-ip-monitor
sudo chown -R ${USER}:${USER} /opt/public-ip-monitor
sudo chmod -R 755 /opt/public-ip-monitor
```

#### 2. Deploy the Application

```bash
# From your development machine, copy the binary to the server
scp ./bin/public-ip-monitor user@your-server:/opt/public-ip-monitor/

# Copy configuration file (if customized)
scp ./config.json user@your-server:/opt/public-ip-monitor/

# SSH into the server and make the binary executable
ssh user@your-server
chmod +x /opt/public-ip-monitor/public-ip-monitor
```

#### 3. Test the Deployment

```bash
# Test the application manually
cd /opt/public-ip-monitor
./public-ip-monitor -check
```

### Systemd Service (Linux)

Create a systemd service for automatic startup and management:

#### 1. Create Service File

Create `/etc/systemd/system/public-ip-monitor.service`:

```ini
[Unit]
Description=Public IP Monitor - Monitors public IP address changes
Documentation=https://github.com/your-repo/public-ip-monitor
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=your-user
Group=your-group
WorkingDirectory=/opt/public-ip-monitor
ExecStart=/opt/public-ip-monitor/public-ip-monitor
ExecReload=/bin/kill -HUP $MAINPID

# Restart policy
Restart=always
RestartSec=10
StartLimitInterval=60
StartLimitBurst=3

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/public-ip-monitor

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=public-ip-monitor

[Install]
WantedBy=multi-user.target
```

#### 2. Enable and Start Service

```bash
# Reload systemd to recognize the new service
sudo systemctl daemon-reload

# Enable the service to start automatically at boot
sudo systemctl enable public-ip-monitor

# Start the service immediately
sudo systemctl start public-ip-monitor

# Verify the service is running
sudo systemctl status public-ip-monitor
```

#### 3. Service Management Commands

```bash
# View service status
sudo systemctl status public-ip-monitor

# View recent logs
sudo journalctl -u public-ip-monitor -f

# View logs from the last hour
sudo journalctl -u public-ip-monitor --since "1 hour ago"

# Restart the service
sudo systemctl restart public-ip-monitor

# Stop the service
sudo systemctl stop public-ip-monitor

# Disable auto-start
sudo systemctl disable public-ip-monitor

# Remove the service (after stopping and disabling)
sudo rm /etc/systemd/system/public-ip-monitor.service
sudo systemctl daemon-reload
```

📝 Changelog
Version 1.0.0

Initial release with core IP monitoring functionality
Email and WhatsApp notification support
Systemd service integration
Cross-platform compilation support
# Public IP Monitor

A professional, modular Go application that monitors your public IP address and sends notifications when it changes. Built with clean architecture principles and fully independent, reusable components.

## Features

🔍 **Continuous IP Monitoring** - Monitors your public IP using multiple services for reliability  
📧 **Email Notifications** - SMTP email alerts with customizable messages  
📱 **WhatsApp Notifications** - Meta Business API integration for instant messaging  
🕐 **Timezone-Aware Logging** - Custom logger with configurable timezone support  
📊 **IP Change History** - Persistent storage and history tracking  
⚙️ **Flexible Configuration** - JSON-based configuration with validation  
🔄 **Graceful Shutdown** - Proper signal handling and cleanup  
🏗️ **Modular Design** - Independent, reusable packages

## Quick Start

### 1. Build the Application

```bash
make build
```

### 2. First Run (Creates Config)

```bash
make run
```

This creates a default `config.json` file. Update it with your credentials.

### 3. Configure Your Settings

Edit `config.json`:

```json
{
    "check_interval_seconds": 300,
    "logging": {
        "timezone": "America/New_York",
        "format": "2006-01-02 15:04:05"
    },
    "email": {
        "enabled": true,
        "from": "your-email@gmail.com",
        "password": "your-app-password",
        "to": "recipient@gmail.com",
        "smtp_host": "smtp.gmail.com",
        "smtp_port": "587"
    },
    "whatsapp": {
        "enabled": true,
        "token": "YOUR_META_BUSINESS_TOKEN",
        "phone_id": "YOUR_PHONE_NUMBER_ID",
        "recipient_number": "1234567890"
    }
}
```

### 4. Start Monitoring

```bash
make run
```

## Architecture

Clean, professional structure with single responsibility principles:

```
public-ip-monitor/
├── cmd/                    # Application entry point
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── ip/                # IP monitoring logic
│   └── logger/            # Custom logging
└── pkg/                   # Reusable packages
    ├── email/             # Email client (fully independent)
    └── whatsapp/          # WhatsApp client (fully independent)
```

## Usage

### Basic Commands

```bash
# Monitor continuously
./bin/public-ip-monitor

# Check IP once and exit
./bin/public-ip-monitor -check

# Show IP change history
./bin/public-ip-monitor -history

# Use custom config file
./bin/public-ip-monitor -config=/path/to/config.json
```

### Make Commands

```bash
make build          # Build the application
make run            # Build and run
make check          # Check IP once
make history        # Show IP history
make test           # Run tests
make clean          # Clean build artifacts
make install        # Install dependencies
```

## Configuration Guide

### Email Setup (Gmail)

1. Enable 2FA on your Google account
2. Generate an App Password:
   - Google Account → Security → 2-Step Verification → App passwords
3. Use the App Password (not your regular password) in config

### WhatsApp Setup

1. Create a [Meta Business Account](https://business.facebook.com/)
2. Set up WhatsApp Business API
3. Get your Phone Number ID and Access Token
4. Add recipient numbers to your business account

### Configuration Options

| Setting | Description | Default |
|---------|-------------|---------|
| `check_interval_seconds` | How often to check IP (seconds) | 300 |
| `logging.timezone` | Timezone for log timestamps | UTC |
| `logging.format` | Time format for logs | 2006-01-02 15:04:05 |
| `email.enabled` | Enable email notifications | true |
| `email.smtp_host` | SMTP server hostname | smtp.gmail.com |
| `email.smtp_port` | SMTP server port | 587 |
| `whatsapp.enabled` | Enable WhatsApp notifications | false |
| `whatsapp.api_version` | Meta API version | v17.0 |
| `ip.services` | List of IP detection services | 3 reliable services |
| `ip.timeout_seconds` | Service request timeout | 30 |

## Reusable Packages

Both email and WhatsApp packages are completely independent and can be used in other Go projects:

### Email Package

```go
import "public-ip-monitor/pkg/email"

factory := email.NewSMTPFactory()
client, err := factory.NewClient(email.Config{
    From:     "sender@example.com",
    Password: "app-password",
    SMTPHost: "smtp.gmail.com",
    SMTPPort: "587",
    Timeout:  30,
})

err = client.Send(ctx, email.Message{
    To:      "recipient@example.com",
    Subject: "Test Subject",
    Body:    "Test Body",
})
```

### WhatsApp Package

```go
import "public-ip-monitor/pkg/whatsapp"

factory := whatsapp.NewMetaFactory()
client, err := factory.NewClient(whatsapp.Config{
    Token:   "your-meta-token",
    PhoneID: "your-phone-id",
    APIVersion: "v17.0",
    TimeoutSeconds: 30,
})

err = client.Send(ctx, whatsapp.Message{
    To:   "1234567890",
    Text: "Hello from Go!",
})
```

## Deployment

### Systemd Service (Linux)

Create `/etc/systemd/system/public-ip-monitor.service`:

```ini
[Unit]
Description=Public IP Monitor
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/opt/public-ip-monitor
ExecStart=/opt/public-ip-monitor/bin/public-ip-monitor
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl enable public-ip-monitor
sudo systemctl start public-ip-monitor
sudo systemctl status public-ip-monitor
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o bin/public-ip-monitor cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/bin/public-ip-monitor .
COPY --from=builder /app/configs/config.example.json config.json
CMD ["./public-ip-monitor"]
```

### Cross-Platform Builds

```bash
make build-linux     # Linux AMD64
make build-windows   # Windows AMD64
make build-macos     # macOS AMD64
make build-all       # All platforms
```

## Development

### Project Structure Benefits

✅ **Modularity** - Each feature is completely independent  
✅ **Reusability** - Email and WhatsApp packages work standalone  
✅ **Testability** - Interface-based design enables easy testing  
✅ **Maintainability** - Single responsibility principle throughout  
✅ **Extensibility** - Easy to add new notification channels or IP services

### Adding New Features

Want to add Slack notifications? Just create `pkg/slack/` following the same pattern:

```go
// pkg/slack/types.go
type Client interface {
    Send(ctx context.Context, message Message) error
    Close() error
}

// pkg/slack/client.go
type SlackClient struct { /* implementation */ }
```

Then use it independently in main.go just like email and WhatsApp.

## Troubleshooting

### Common Issues

**"Config file not found"**
- Run the application once to generate default config
- Ensure you're in the correct directory

**"Email authentication failed"**
- Use App Password for Gmail (not regular password)
- Ensure 2FA is enabled on Google account
- Check SMTP settings

**"WhatsApp API errors"**
- Verify Meta Business account setup
- Check token permissions and expiration
- Ensure recipient number is registered with business account

**"Network timeouts"**
- Check internet connectivity
- Verify firewall settings
- Increase timeout values in config

### Debug Mode

Enable detailed logging by modifying the logger configuration or add debug statements as needed.

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes following the existing architecture
4. Add tests for new functionality
5. Submit a pull request

## License

This project is open source. Feel free to use, modify, and distribute.

## Architecture Philosophy

This project demonstrates professional Go development practices:

- **Clean Architecture** - Clear separation between business logic and infrastructure
- **Interface-Based Design** - Easy testing and swapping of implementations
- **Configuration-Driven** - Behavior controlled through configuration, not code
- **Error Handling** - Comprehensive error handling with proper context
- **Graceful Degradation** - Continues operating even if some features fail
- **Production Ready** - Logging, monitoring, and deployment considerations

The codebase serves as an excellent example of how to structure Go applications that are maintainable, testable, and scalable.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"public-ip-monitor/internal/config"
	"public-ip-monitor/internal/ip"
	"public-ip-monitor/internal/logger"
	"public-ip-monitor/pkg/email"
	"public-ip-monitor/pkg/whatsapp"
)

func main() {
	// Parse command line flags
	var (
		configPath  = flag.String("config", "config.json", "Path to configuration file")
		showHistory = flag.Bool("history", false, "Show IP change history and exit")
		checkOnce   = flag.Bool("check", false, "Check IP once and exit")
	)
	flag.Parse()

	// Load configuration
	configManager := config.NewManager(*configPath)
	cfg, err := configManager.Load()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log, err := logger.New(cfg.Logging)
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	log.Info("Starting Public IP Monitor...")

	// Initialize IP storage
	storage := ip.NewStorage(cfg.IP.DataDir, cfg.IP.RecordsFile, cfg.IP.LastIPFile)
	if err := storage.Initialize(); err != nil {
		log.Errorf("Failed to initialize storage: %v", err)
		os.Exit(1)
	}

	// Initialize IP fetcher
	fetcher := ip.NewFetcher(cfg.IP.Services, cfg.IP.TimeoutSeconds)

	// Handle history command
	if *showHistory {
		monitor := ip.NewMonitor(fetcher, storage, nil)
		if err := monitor.PrintHistory(); err != nil {
			log.Errorf("Failed to print history: %v", err)
			os.Exit(1)
		}
		return
	}

	// Initialize email client (independent)
	var emailClient email.Client
	if cfg.Email.Enabled {
		emailFactory := email.NewSMTPFactory()
		emailConfig := email.Config{
			From:     cfg.Email.From,
			Password: cfg.Email.Password,
			SMTPHost: cfg.Email.SMTPHost,
			SMTPPort: cfg.Email.SMTPPort,
			Timeout:  cfg.Email.Timeout,
		}
		emailClient, err = emailFactory.NewClient(emailConfig)
		if err != nil {
			log.Errorf("Failed to create email client: %v", err)
			os.Exit(1)
		}
		defer emailClient.Close()
		log.Info("Email notifications enabled")
	} else {
		log.Info("Email notifications disabled")
	}

	// Initialize WhatsApp client (independent)
	var whatsappClient whatsapp.Client
	if cfg.WhatsApp.Enabled {
		whatsappFactory := whatsapp.NewMetaFactory()
		whatsappConfig := whatsapp.Config{
			Token:          cfg.WhatsApp.Token,
			PhoneID:        cfg.WhatsApp.PhoneID,
			APIVersion:     cfg.WhatsApp.APIVersion,
			TimeoutSeconds: cfg.WhatsApp.TimeoutSeconds,
		}
		whatsappClient, err = whatsappFactory.NewClient(whatsappConfig)
		if err != nil {
			log.Errorf("Failed to create WhatsApp client: %v", err)
			os.Exit(1)
		}
		defer whatsappClient.Close()
		log.Info("WhatsApp notifications enabled")
	} else {
		log.Info("WhatsApp notifications disabled")
	}

	// Create IP change handler with independent notification sending
	changeHandler := func(oldIP, newIP string) error {
		if oldIP == "" {
			oldIP = "Unknown"
		}

		log.Infof("IP changed from %s to %s", oldIP, newIP)

		// Prepare notification data
		timestamp := time.Now().Format("2006-01-02 15:04:05")

		// Send email notification (independent)
		if cfg.Email.Enabled && emailClient != nil {
			emailSubject := "🚨 IP Address Changed - Public IP Monitor"
			emailBody := fmt.Sprintf(`IP Address Change Notification

Your public IP address has changed:

Previous IP: %s
New IP: %s
Change Time: %s

This notification was sent automatically by your IP monitoring service.

Best regards,
Public IP Monitor`, oldIP, newIP, timestamp)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			emailMsg := email.Message{
				To:      cfg.Email.To,
				Subject: emailSubject,
				Body:    emailBody,
			}

			if err := emailClient.Send(ctx, emailMsg); err != nil {
				log.Errorf("Failed to send email notification: %v", err)
			} else {
				log.Info("Email notification sent successfully")
			}
			cancel()
		}

		// Send WhatsApp notification (independent)
		if cfg.WhatsApp.Enabled && whatsappClient != nil {
			whatsappMessage := fmt.Sprintf("🚨 IP Address Changed!\n\nOld IP: %s\nNew IP: %s\nTime: %s\n\nPublic IP Monitor",
				oldIP, newIP, timestamp)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
			whatsappMsg := whatsapp.Message{
				To:   cfg.WhatsApp.RecipientNumber,
				Text: whatsappMessage,
			}

			if err := whatsappClient.Send(ctx, whatsappMsg); err != nil {
				log.Errorf("Failed to send WhatsApp notification: %v", err)
			} else {
				log.Info("WhatsApp notification sent successfully")
			}
			cancel()
		}

		return nil
	}

	// Initialize IP monitor
	monitor := ip.NewMonitor(fetcher, storage, changeHandler)

	// Handle check-once command
	if *checkOnce {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		result := monitor.CheckOnce(ctx)
		if result.Error != nil {
			log.Errorf("Check failed: %v", result.Error)
			os.Exit(1)
		}

		if result.Changed {
			log.Infof("IP changed from %s to %s", result.LastIP, result.CurrentIP)
		} else {
			log.Infof("IP unchanged: %s", result.CurrentIP)
		}
		return
	}

	// Get last known IP for logging
	lastIP, err := storage.ReadLastIP()
	if err != nil {
		log.Errorf("Failed to read last IP: %v", err)
	} else if lastIP == "" {
		log.Info("No last IP found - this appears to be the first run")
	} else {
		log.Infof("Last known IP: %s", lastIP)
	}

	// Start monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Infof("Starting IP monitoring every %d seconds...", cfg.CheckIntervalSeconds)
	resultChan := monitor.StartMonitoring(ctx, config.GetCheckInterval(cfg))

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Main monitoring loop
	for {
		select {
		case result, ok := <-resultChan:
			if !ok {
				log.Info("Monitoring stopped")
				return
			}

			if result.Error != nil {
				log.Errorf("IP check failed: %v", result.Error)
				continue
			}

			if result.Changed {
				log.Infof("IP changed from %s to %s", result.LastIP, result.CurrentIP)
			} else {
				log.Infof("IP unchanged: %s", result.CurrentIP)
			}

		case sig := <-sigChan:
			log.Infof("Received signal %v, shutting down gracefully...", sig)
			cancel()

			// Give some time for cleanup
			time.Sleep(1 * time.Second)
			log.Info("Shutdown complete")
			return
		}
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
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

	// Pre-allocate channels for notifications to avoid blocking
	notificationChan := make(chan notificationRequest, 10) // Buffered channel

	// Start notification worker goroutine
	go notificationWorker(notificationChan, emailClient, whatsappClient, cfg, log)

	// Create IP change handler with async notifications
	changeHandler := func(oldIP, newIP string) error {
		if oldIP == "" {
			oldIP = "Unknown"
		}

		log.Infof("IP changed from %s to %s", oldIP, newIP)

		// Send notification request asynchronously
		select {
		case notificationChan <- notificationRequest{
			OldIP:     oldIP,
			NewIP:     newIP,
			Timestamp: time.Now(),
		}:
			// Notification queued successfully
		default:
			// Channel full, log warning but don't block
			log.Warn("Notification channel full, dropping notification")
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

		// Wait for any pending notifications before exit
		close(notificationChan)
		time.Sleep(100 * time.Millisecond)
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
				close(notificationChan) // Close notification channel
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

			// Close notification channel and wait for worker to finish
			close(notificationChan)
			time.Sleep(2 * time.Second) // Give time for pending notifications

			log.Info("Shutdown complete")
			return
		}
	}
}

// notificationRequest represents a notification to be sent
type notificationRequest struct {
	OldIP     string
	NewIP     string
	Timestamp time.Time
}

// notificationWorker processes notifications asynchronously
func notificationWorker(
	notificationChan <-chan notificationRequest,
	emailClient email.Client,
	whatsappClient whatsapp.Client,
	cfg *config.Config,
	log *logger.Logger,
) {
	// Set GOMAXPROCS for better CPU utilization in containers
	if runtime.GOMAXPROCS(0) == 1 {
		runtime.GOMAXPROCS(2) // Minimum 2 for concurrent notifications
	}

	for req := range notificationChan {
		// Process notifications concurrently
		var wg sync.WaitGroup

		// Send email notification (if enabled)
		if cfg.Email.Enabled && emailClient != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sendEmailNotification(emailClient, cfg, req, log)
			}()
		}

		// Send WhatsApp notification (if enabled)
		if cfg.WhatsApp.Enabled && whatsappClient != nil {
			wg.Add(1)
			go func() {
				defer wg.Done()
				sendWhatsAppNotification(whatsappClient, cfg, req, log)
			}()
		}

		// Wait for all notifications to complete (with timeout)
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// All notifications completed
		case <-time.After(30 * time.Second):
			// Timeout waiting for notifications
			log.Warn("Notification timeout - some notifications may not have completed")
		}
	}
}

// sendEmailNotification sends email notification with retry logic
func sendEmailNotification(
	client email.Client,
	cfg *config.Config,
	req notificationRequest,
	log *logger.Logger,
) {
	emailSubject := config.BuildEmailSubject()
	emailBody := config.BuildEmailBody(req.OldIP, req.NewIP, req.Timestamp)

	// Retry logic with exponential backoff
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		emailMsg := email.Message{
			To:      cfg.Email.To,
			Subject: emailSubject,
			Body:    emailBody,
		}

		if err := client.Send(ctx, emailMsg); err != nil {
			cancel()
			if attempt == maxRetries {
				log.Errorf("Failed to send email notification after %d attempts: %v", maxRetries, err)
				return
			}

			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			log.Warnf("Email notification attempt %d failed, retrying in %v: %v", attempt, backoff, err)
			time.Sleep(backoff)
			continue
		}

		cancel()
		log.Info("Email notification sent successfully")
		return
	}
}

// sendWhatsAppNotification sends WhatsApp notification with retry logic
func sendWhatsAppNotification(
	client whatsapp.Client,
	cfg *config.Config,
	req notificationRequest,
	log *logger.Logger,
) {
	whatsappMessage := config.BuildWhatsAppMessage(req.OldIP, req.NewIP, req.Timestamp)

	// Retry logic with exponential backoff
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		whatsappMsg := whatsapp.Message{
			To:   cfg.WhatsApp.RecipientNumber,
			Text: whatsappMessage,
		}

		if err := client.Send(ctx, whatsappMsg); err != nil {
			cancel()
			if attempt == maxRetries {
				log.Errorf("Failed to send WhatsApp notification after %d attempts: %v", maxRetries, err)
				return
			}

			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			log.Warnf("WhatsApp notification attempt %d failed, retrying in %v: %v", attempt, backoff, err)
			time.Sleep(backoff)
			continue
		}

		cancel()
		log.Info("WhatsApp notification sent successfully")
		return
	}
}

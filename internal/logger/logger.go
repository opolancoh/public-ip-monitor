package logger

import (
	"fmt"
	"log"
	"time"

	"public-ip-monitor/internal/config"
)

// Logger handles logging with timezone support
type Logger struct {
	timezone *time.Location
	format   string
}

// New creates a new logger with timezone configuration
func New(cfg config.LoggingConfig) (*Logger, error) {
	timezone, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", cfg.Timezone, err)
	}

	return &Logger{
		timezone: timezone,
		format:   cfg.Format,
	}, nil
}

// Info logs an info message with timestamp and timezone
func (l *Logger) Info(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	log.Printf("[INFO] %s - %s", timestamp, message)
}

// Error logs an error message with timestamp and timezone
func (l *Logger) Error(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	log.Printf("[ERROR] %s - %s", timestamp, message)
}

// Warn logs a warning message with timestamp and timezone
func (l *Logger) Warn(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	log.Printf("[WARN] %s - %s", timestamp, message)
}

// Debug logs a debug message with timestamp and timezone
func (l *Logger) Debug(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	log.Printf("[DEBUG] %s - %s", timestamp, message)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.Info(fmt.Sprintf(format, args...))
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Error(fmt.Sprintf(format, args...))
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Debug(fmt.Sprintf(format, args...))
}

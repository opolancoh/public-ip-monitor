package logger

import (
	"fmt"
	"log"
	"os"
	"time"

	"public-ip-monitor/internal/config"
)

// Logger handles logging with timezone support
type Logger struct {
	timezone   *time.Location
	format     string
	identifier string // New field for log identifier
	logger     *log.Logger
}

// New creates a new logger with timezone configuration
func New(cfg config.LoggingConfig) (*Logger, error) {
	timezone, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone %s: %w", cfg.Timezone, err)
	}

	return &Logger{
		timezone:   timezone,
		format:     cfg.Format,
		identifier: cfg.Identifier,
		logger:     log.New(os.Stdout, "", 0),
	}, nil
}

func (l *Logger) Info(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	l.logger.Printf("[%s] [INFO] %s - %s", l.identifier, timestamp, message)
}

func (l *Logger) Error(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	l.logger.Printf("[%s] [ERROR] %s - %s", l.identifier, timestamp, message)
}

func (l *Logger) Warn(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	l.logger.Printf("[%s] [WARN] %s - %s", l.identifier, timestamp, message)
}

func (l *Logger) Debug(message string) {
	timestamp := time.Now().In(l.timezone).Format(l.format + " MST")
	l.logger.Printf("[%s] [DEBUG] %s - %s", l.identifier, timestamp, message)
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

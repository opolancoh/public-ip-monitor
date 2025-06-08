package ip

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DataFilePerm = 0644
)

// Record represents an IP change record
type Record struct {
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
}

// Storage handles IP data persistence
type Storage struct {
	dataDir     string
	recordsFile string
	lastIPFile  string
}

// NewStorage creates a new IP storage
func NewStorage(dataDir, recordsFile, lastIPFile string) *Storage {
	return &Storage{
		dataDir:     dataDir,
		recordsFile: filepath.Join(dataDir, recordsFile),
		lastIPFile:  filepath.Join(dataDir, lastIPFile),
	}
}

// Initialize creates the data directory if it doesn't exist
func (s *Storage) Initialize() error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	return nil
}

// ReadLastIP reads the last known IP from file
func (s *Storage) ReadLastIP() (string, error) {
	data, err := os.ReadFile(s.lastIPFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // File doesn't exist, return empty string
		}
		return "", fmt.Errorf("failed to read last IP file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// SaveLastIP saves the current IP to file
func (s *Storage) SaveLastIP(ip string) error {
	if err := s.Initialize(); err != nil {
		return err
	}

	if err := os.WriteFile(s.lastIPFile, []byte(ip), DataFilePerm); err != nil {
		return fmt.Errorf("failed to save last IP: %w", err)
	}
	return nil
}

// SaveRecord adds a new IP change record
func (s *Storage) SaveRecord(ip string) error {
	if err := s.Initialize(); err != nil {
		return err
	}

	record := Record{
		IP:        ip,
		Timestamp: time.Now(),
	}

	// Read existing records
	records, err := s.GetHistory()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing records: %w", err)
	}

	// Add new record
	records = append(records, record)

	// Save updated records
	data, err := json.MarshalIndent(records, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal records: %w", err)
	}

	if err := os.WriteFile(s.recordsFile, data, DataFilePerm); err != nil {
		return fmt.Errorf("failed to save IP record: %w", err)
	}

	return nil
}

// GetHistory returns the history of IP changes
func (s *Storage) GetHistory() ([]Record, error) {
	var records []Record

	data, err := os.ReadFile(s.recordsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return records, nil // File doesn't exist, return empty slice
		}
		return nil, fmt.Errorf("failed to read records file: %w", err)
	}

	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal records: %w", err)
	}

	return records, nil
}

// GetHistoryCount returns the number of IP change records
func (s *Storage) GetHistoryCount() (int, error) {
	records, err := s.GetHistory()
	if err != nil {
		return 0, err
	}
	return len(records), nil
}

// ClearHistory removes all IP change records (useful for testing or cleanup)
func (s *Storage) ClearHistory() error {
	if err := os.Remove(s.recordsFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear history: %w", err)
	}
	return nil
}

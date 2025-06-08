package ip

import (
	"context"
	"fmt"
	"time"
)

// ChangeHandler is called when IP changes are detected
type ChangeHandler func(oldIP, newIP string) error

// Monitor handles IP monitoring logic
type Monitor struct {
	fetcher *Fetcher
	storage *Storage
	handler ChangeHandler
}

// NewMonitor creates a new IP monitor
func NewMonitor(fetcher *Fetcher, storage *Storage, handler ChangeHandler) *Monitor {
	return &Monitor{
		fetcher: fetcher,
		storage: storage,
		handler: handler,
	}
}

// CheckResult represents the result of an IP check
type CheckResult struct {
	CurrentIP string
	LastIP    string
	Changed   bool
	Error     error
}

// CheckOnce performs a single IP check
func (m *Monitor) CheckOnce(ctx context.Context) CheckResult {
	// Get current IP
	currentIP, err := m.fetcher.GetCurrentIP(ctx)
	if err != nil {
		return CheckResult{Error: fmt.Errorf("failed to get current IP: %w", err)}
	}

	// Get last known IP
	lastIP, err := m.storage.ReadLastIP()
	if err != nil {
		return CheckResult{Error: fmt.Errorf("failed to read last IP: %w", err)}
	}

	// Check if IP has changed
	changed := currentIP != lastIP

	result := CheckResult{
		CurrentIP: currentIP,
		LastIP:    lastIP,
		Changed:   changed,
	}

	if changed {
		// Handle IP change
		if err := m.handleIPChange(lastIP, currentIP); err != nil {
			result.Error = fmt.Errorf("failed to handle IP change: %w", err)
			return result
		}
	}

	return result
}

// StartMonitoring starts continuous IP monitoring
func (m *Monitor) StartMonitoring(ctx context.Context, interval time.Duration) <-chan CheckResult {
	resultChan := make(chan CheckResult, 1)

	go func() {
		defer close(resultChan)

		// Check immediately on startup
		select {
		case resultChan <- m.CheckOnce(ctx):
		case <-ctx.Done():
			return
		}

		// Set up periodic checking
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				select {
				case resultChan <- m.CheckOnce(ctx):
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return resultChan
}

// handleIPChange processes an IP change
func (m *Monitor) handleIPChange(oldIP, newIP string) error {
	// Save new IP
	if err := m.storage.SaveLastIP(newIP); err != nil {
		return fmt.Errorf("failed to save new IP: %w", err)
	}

	// Save record
	if err := m.storage.SaveRecord(newIP); err != nil {
		return fmt.Errorf("failed to save IP record: %w", err)
	}

	// Call change handler if provided
	if m.handler != nil {
		if err := m.handler(oldIP, newIP); err != nil {
			return fmt.Errorf("change handler failed: %w", err)
		}
	}

	return nil
}

// GetHistory returns IP change history
func (m *Monitor) GetHistory() ([]Record, error) {
	return m.storage.GetHistory()
}

// PrintHistory prints the IP change history to console
func (m *Monitor) PrintHistory() error {
	records, err := m.GetHistory()
	if err != nil {
		return fmt.Errorf("failed to get IP history: %w", err)
	}

	if len(records) == 0 {
		fmt.Println("\n=== IP Change History ===")
		fmt.Println("No IP changes recorded yet.")
		fmt.Println("========================")
		return nil
	}

	fmt.Println("\n=== IP Change History ===")
	for i, record := range records {
		fmt.Printf("%d. IP: %s - Time: %s\n",
			i+1, record.IP, record.Timestamp.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("========================")

	return nil
}

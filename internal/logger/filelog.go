package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	logFileMutex sync.Mutex
)

// CheckLogEntry represents a single check log entry
type CheckLogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	TargetID     int                    `json:"target_id"`
	TargetName   string                 `json:"target_name"`
	Type         string                 `json:"type"`
	Address      string                 `json:"address"`
	Status       string                 `json:"status"`
	ResponseTime int64                  `json:"response_time"`
	Message      string                 `json:"message"`
	Request      map[string]interface{} `json:"request,omitempty"`
	Response     map[string]interface{} `json:"response,omitempty"`
}

// InitLogFileLog initializes file-based logging for check results
func InitLogFileLog(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	return nil
}

// WriteCheckLog writes a check result to the log file
func WriteCheckLog(logDir string, entry *CheckLogEntry) error {
	logFileMutex.Lock()
	defer logFileMutex.Unlock()

	// Create log file path with date: logs/check-2026-01-14.jsonl
	date := time.Now().Format("2006-01-02")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("check-%s.jsonl", date))

	// Open file in append mode
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	// Set timestamp
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Write to file with newline
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// LogQueryRequest represents a log query request
type LogQueryRequest struct {
	TargetID   *int       `json:"target_id,omitempty"`
	Status     string     `json:"status,omitempty"`
	StartTime  *time.Time `json:"start_time,omitempty"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// LogQueryResult represents the result of a log query
type LogQueryResult struct {
	Total  int              `json:"total"`
	Logs   []*CheckLogEntry `json:"logs"`
}

// QueryCheckLogs queries check logs from files
func QueryCheckLogs(logDir string, req *LogQueryRequest) (*LogQueryResult, error) {
	result := &LogQueryResult{
		Logs: make([]*CheckLogEntry, 0),
	}

	// Determine date range for log files
	var startDate, endDate time.Time
	if req.StartTime != nil {
		startDate = *req.StartTime
	} else {
		startDate = time.Now().AddDate(0, 0, -7) // Default: last 7 days
	}

	if req.EndTime != nil {
		endDate = *req.EndTime
	} else {
		endDate = time.Now()
	}

	// Iterate through each day in the range
	matchedEntries := make([]*CheckLogEntry, 0)
	for d := startDate; d.Before(endDate.AddDate(0, 0, 1)); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		logFilePath := filepath.Join(logDir, fmt.Sprintf("check-%s.jsonl", dateStr))

		// Check if file exists
		if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
			continue
		}

		// Read and parse log file
		entries, err := readLogFile(logFilePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		// Filter entries based on query criteria
		for _, entry := range entries {
			if !matchesQuery(entry, req) {
				continue
			}
			matchedEntries = append(matchedEntries, entry)
		}
	}

	// Sort by timestamp descending (newest first)
	sortEntries(matchedEntries)

	// Apply pagination
	result.Total = len(matchedEntries)
	if req.Limit <= 0 {
		req.Limit = 100 // Default limit
	}

	start := req.Offset
	if start > len(matchedEntries) {
		start = len(matchedEntries)
	}

	end := start + req.Limit
	if end > len(matchedEntries) {
		end = len(matchedEntries)
	}

	if start < end {
		result.Logs = matchedEntries[start:end]
	}

	return result, nil
}

// readLogFile reads a log file and returns its entries
func readLogFile(logFilePath string) ([]*CheckLogEntry, error) {
	file, err := os.Open(logFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := make([]*CheckLogEntry, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var entry CheckLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip invalid lines
		}

		entries = append(entries, &entry)
	}

	return entries, scanner.Err()
}

// matchesQuery checks if an entry matches the query criteria
func matchesQuery(entry *CheckLogEntry, req *LogQueryRequest) bool {
	// Filter by target_id
	if req.TargetID != nil && entry.TargetID != *req.TargetID {
		return false
	}

	// Filter by status
	if req.Status != "" && entry.Status != req.Status {
		return false
	}

	// Filter by time range
	if req.StartTime != nil && entry.Timestamp.Before(*req.StartTime) {
		return false
	}

	if req.EndTime != nil && entry.Timestamp.After(*req.EndTime) {
		return false
	}

	return true
}

// sortEntries sorts entries by timestamp (newest first)
func sortEntries(entries []*CheckLogEntry) {
	// Simple bubble sort (for small datasets)
	n := len(entries)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if entries[j].Timestamp.Before(entries[j+1].Timestamp) {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}
}

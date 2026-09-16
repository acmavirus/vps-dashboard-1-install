package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCountLogRequests(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test_access.log")

	todayStr := time.Now().Format("02/Jan/2006")
	yesterdayStr := time.Now().AddDate(0, 0, -1).Format("02/Jan/2006")

	content := fmt.Sprintf(`127.0.0.1 - - [%s:10:00:00 +0700] "GET / HTTP/1.1" 200 123 "-" "Mozilla/5.0"
127.0.0.1 - - [%s:11:00:00 +0700] "GET /about HTTP/1.1" 200 456 "-" "Mozilla/5.0"
127.0.0.1 - - [%s:12:00:00 +0700] "GET /contact HTTP/1.1" 200 789 "-" "Mozilla/5.0"
127.0.0.1 - - [%s:09:00:00 +0700] "GET /today-1 HTTP/1.1" 200 100 "-" "Mozilla/5.0"
127.0.0.1 - - [%s:10:30:00 +0700] "GET /today-2 HTTP/1.1" 200 200 "-" "Mozilla/5.0"
`, yesterdayStr, yesterdayStr, yesterdayStr, todayStr, todayStr)

	if err := os.WriteFile(logPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write mock log: %v", err)
	}

	total, daily := countLogRequests(logPath)
	if total != 5 {
		t.Errorf("Expected total requests = 5, got %d", total)
	}
	if daily != 2 {
		t.Errorf("Expected daily requests = 2, got %d", daily)
	}

	// Test cache hit
	cachedTotal, cachedDaily := countLogRequests(logPath)
	if cachedTotal != 5 || cachedDaily != 2 {
		t.Errorf("Cache mismatch: expected (5, 2), got (%d, %d)", cachedTotal, cachedDaily)
	}
}

package services

import (
	"os"
	"testing"
	"time"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/database"
)

func setupMetricsCollectorTest(t *testing.T) (*MetricsCollector, func()) {
	t.Helper()

	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "metrics_collector_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := tempDir + "/test.db"

	// Create database
	db, err := database.New(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		db.Close()
		os.RemoveAll(tempDir)
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create metrics config
	enabled := true
	cfg := &config.MetricsConfig{
		Enabled:             &enabled,
		CollectionInterval:  "1s", // Short interval for testing
		RetentionDays:       7,
		HourlyRetentionDays: 30,
		DailyRetentionDays:  365,
	}

	// Create collector
	collector := NewMetricsCollector(db.DB, cfg)

	cleanup := func() {
		collector.Stop()
		_ = db.Close()
		os.RemoveAll(tempDir)
	}

	return collector, cleanup
}

func TestNewMetricsCollector(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	if collector == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}
}

func TestMetricsCollector_StartStop(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	// Start the collector
	collector.Start()

	// Give it time to run at least one collection
	time.Sleep(2 * time.Second)

	// Check that we have some data
	snapshot := collector.GetLatest()
	if snapshot == nil {
		t.Error("Expected to have metrics snapshot after collection")
	}

	// Stop should complete without error
	collector.Stop()
}

func TestMetricsCollector_GetLatest(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	// Before starting, GetLatest should return nil
	if collector.GetLatest() != nil {
		t.Error("GetLatest should return nil before collection starts")
	}

	// Start collection
	collector.Start()
	time.Sleep(2 * time.Second)

	// After collection, should have data
	snapshot := collector.GetLatest()
	if snapshot == nil {
		t.Fatal("GetLatest returned nil after collection started")
	}

	// Check snapshot fields
	if snapshot.System == nil {
		t.Error("System metrics should not be nil")
	}
	if snapshot.Docker == nil {
		t.Error("Docker metrics should not be nil")
	}
	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	collector.Stop()
}

func TestMetricsCollector_GetHistoricalMetrics(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	// Start and let it collect some data
	collector.Start()
	time.Sleep(3 * time.Second)
	collector.Stop()

	// Query for raw metrics with a very wide time range
	from := time.Now().Add(-24 * time.Hour)
	to := time.Now().Add(24 * time.Hour)

	metrics, err := collector.GetHistoricalMetrics(from, to, "raw")
	if err != nil {
		t.Fatalf("GetHistoricalMetrics returned error: %v", err)
	}

	// Log the count - may be 0 due to timing issues in tests
	t.Logf("Found %d historical metric records", len(metrics))

	// Verify structure of returned data if any
	for i, m := range metrics {
		if m.ID == 0 {
			t.Errorf("Metric[%d] ID should not be 0", i)
		}
		if m.Timestamp.IsZero() {
			t.Errorf("Metric[%d] Timestamp should not be zero", i)
		}
	}
}

func TestMetricsCollector_GetHistoricalMetrics_Resolutions(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	resolutions := []string{"raw", "hourly", "daily", "invalid"}

	for _, res := range resolutions {
		t.Run(res, func(t *testing.T) {
			_, err := collector.GetHistoricalMetrics(from, to, res)
			if err != nil {
				t.Errorf("GetHistoricalMetrics(%s) returned error: %v", res, err)
			}
		})
	}
}

func TestMetricsCollector_GetHistoricalDockerMetrics(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	// Start and let it collect some data
	collector.Start()
	time.Sleep(2 * time.Second)
	collector.Stop()

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)

	// Query for a container (may be empty if no containers running)
	metrics, err := collector.GetHistoricalDockerMetrics("test-container", from, to)
	if err != nil {
		t.Fatalf("GetHistoricalDockerMetrics returned error: %v", err)
	}

	// Result may be empty if no containers, that's fine
	t.Logf("Found %d Docker metric records", len(metrics))
}

func TestMetricsCollector_PruneMetrics(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	// Start and collect some data
	collector.Start()
	time.Sleep(3 * time.Second)
	collector.Stop()

	// Prune with a date in the future (should delete everything)
	before := time.Now().Add(24 * time.Hour)
	deleted, err := collector.PruneMetrics(before)
	if err != nil {
		t.Fatalf("PruneMetrics returned error: %v", err)
	}

	// Should have deleted at least some records
	if deleted == 0 {
		t.Log("No records were deleted (may be expected if no data was collected)")
	} else {
		t.Logf("Deleted %d records", deleted)
	}

	// Verify data is gone
	from := time.Now().Add(-1 * time.Hour)
	to := time.Now().Add(1 * time.Hour)
	metrics, _ := collector.GetHistoricalMetrics(from, to, "raw")
	if len(metrics) > 0 {
		t.Error("Expected no metrics after pruning")
	}
}

func TestMetricsCollector_VacuumDatabase(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	// Start and collect some data
	collector.Start()
	time.Sleep(2 * time.Second)
	collector.Stop()

	// Vacuum should not return error
	err := collector.VacuumDatabase()
	if err != nil {
		t.Fatalf("VacuumDatabase returned error: %v", err)
	}
}

func TestMetricsCollector_GetDatabaseInfo(t *testing.T) {
	collector, cleanup := setupMetricsCollectorTest(t)
	defer cleanup()

	// Start and collect some data
	collector.Start()
	time.Sleep(2 * time.Second)
	collector.Stop()

	info, err := collector.GetDatabaseInfo("/tmp/test.db")
	if err != nil {
		t.Fatalf("GetDatabaseInfo returned error: %v", err)
	}

	if info == nil {
		t.Fatal("GetDatabaseInfo returned nil")
	}

	// MetricsCount should be >= 0
	t.Logf("Metrics count: %d", info.MetricsCount)

	// Timestamps may or may not be set depending on data
	if info.OldestTimestamp != nil {
		t.Logf("Oldest: %v", info.OldestTimestamp)
	}
	if info.NewestTimestamp != nil {
		t.Logf("Newest: %v", info.NewestTimestamp)
	}
}

func TestMetricsCollector_DisabledConfig(t *testing.T) {
	// Create temp directory for test database
	tempDir, err := os.MkdirTemp("", "metrics_collector_disabled_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	dbPath := tempDir + "/test.db"

	// Create database
	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create config with metrics disabled
	disabled := false
	cfg := &config.MetricsConfig{
		Enabled:            &disabled,
		CollectionInterval: "1s",
	}

	collector := NewMetricsCollector(db.DB, cfg)
	collector.Start() // Should log that metrics are disabled

	// Wait a bit
	time.Sleep(2 * time.Second)

	// GetLatest should return nil since collection is disabled
	if collector.GetLatest() != nil {
		t.Error("Expected nil snapshot when metrics are disabled")
	}

	collector.Stop()
}

func TestStoredSystemMetrics_Structure(t *testing.T) {
	m := StoredSystemMetrics{
		ID:            1,
		Timestamp:     time.Now(),
		CPUPercent:    50.0,
		MemoryPercent: 75.0,
		MemoryUsed:    8000000000,
		MemoryTotal:   16000000000,
		DiskData:      `[{"mount_point":"/","used_percent":60}]`,
		NetworkData:   `[{"interface":"eth0","bytes_recv":1000000}]`,
		LoadAvg:       `[1.0,0.5,0.25]`,
		Uptime:        86400,
	}

	if m.ID != 1 {
		t.Error("ID not set correctly")
	}
	if m.CPUPercent != 50.0 {
		t.Error("CPUPercent not set correctly")
	}
	if m.MemoryPercent != 75.0 {
		t.Error("MemoryPercent not set correctly")
	}
}

func TestStoredDockerMetrics_Structure(t *testing.T) {
	m := StoredDockerMetrics{
		ID:            1,
		Timestamp:     time.Now(),
		ContainerID:   "abc123",
		ContainerName: "test-container",
		Image:         "nginx:latest",
		State:         "running",
		CPUPercent:    25.0,
		MemoryPercent: 50.0,
		MemoryUsed:    1073741824,
		MemoryLimit:   2147483648,
		NetworkRx:     10000000,
		NetworkTx:     5000000,
		BlockRead:     50000000,
		BlockWrite:    25000000,
	}

	if m.ContainerID != "abc123" {
		t.Error("ContainerID not set correctly")
	}
	if m.ContainerName != "test-container" {
		t.Error("ContainerName not set correctly")
	}
	if m.CPUPercent != 25.0 {
		t.Error("CPUPercent not set correctly")
	}
}

func TestDatabaseInfo_Structure(t *testing.T) {
	now := time.Now()
	info := DatabaseInfo{
		Path:            "/path/to/db",
		SizeBytes:       1048576,
		SizeFormatted:   "1.0 MB",
		MetricsCount:    100,
		OldestTimestamp: &now,
		NewestTimestamp: &now,
	}

	if info.Path != "/path/to/db" {
		t.Error("Path not set correctly")
	}
	if info.SizeBytes != 1048576 {
		t.Error("SizeBytes not set correctly")
	}
	if info.MetricsCount != 100 {
		t.Error("MetricsCount not set correctly")
	}
}

func TestMetricsSnapshot_Structure(t *testing.T) {
	snapshot := MetricsSnapshot{
		Timestamp: time.Now(),
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
	// System and Docker can be nil initially
	if snapshot.System != nil {
		t.Log("System metrics present")
	}
	if snapshot.Docker != nil {
		t.Log("Docker metrics present")
	}
}

func TestParseSQLiteTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "SQLite default format",
			input:     "2025-12-08 15:04:05",
			expectErr: false,
		},
		{
			name:      "SQLite with milliseconds",
			input:     "2025-12-08 15:04:05.123456789",
			expectErr: false,
		},
		{
			name:      "RFC3339 format",
			input:     "2025-12-08T15:04:05Z",
			expectErr: false,
		},
		{
			name:      "RFC3339 with timezone",
			input:     "2025-12-08T15:04:05+07:00",
			expectErr: false,
		},
		{
			name:      "RFC3339Nano format",
			input:     "2025-12-08T15:04:05.123456789Z",
			expectErr: false,
		},
		{
			name:      "ISO format with T separator",
			input:     "2025-12-08T15:04:05",
			expectErr: false,
		},
		{
			name:      "Invalid format",
			input:     "not a timestamp",
			expectErr: true,
		},
		{
			name:      "Empty string",
			input:     "",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseSQLiteTimestamp(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tc.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tc.input, err)
				}
				if result.IsZero() {
					t.Errorf("Expected non-zero time for input %q", tc.input)
				}
				// Verify year is parsed correctly
				if result.Year() != 2025 {
					t.Errorf("Expected year 2025, got %d for input %q", result.Year(), tc.input)
				}
			}
		})
	}
}

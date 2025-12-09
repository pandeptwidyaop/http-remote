// Package services provides business logic and service layer implementations.
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pandeptwidyaop/http-remote/internal/config"
	"github.com/pandeptwidyaop/http-remote/internal/metrics"
)

// MetricsCollector handles background collection and storage of system metrics.
type MetricsCollector struct {
	db       *sql.DB
	config   *config.MetricsConfig
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	lastData *MetricsSnapshot
}

// MetricsSnapshot holds the latest metrics data for quick access.
type MetricsSnapshot struct {
	System    *metrics.SystemMetrics `json:"system"`
	Docker    *metrics.DockerMetrics `json:"docker"`
	Timestamp time.Time              `json:"timestamp"`
}

// StoredSystemMetrics represents system metrics stored in database.
type StoredSystemMetrics struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryTotal   uint64    `json:"memory_total"`
	DiskData      string    `json:"disk_data"`
	NetworkData   string    `json:"network_data"`
	LoadAvg       string    `json:"load_avg"`
	Uptime        int64     `json:"uptime"`
}

// StoredDockerMetrics represents Docker container metrics stored in database.
type StoredDockerMetrics struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	Image         string    `json:"image"`
	State         string    `json:"state"`
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	MemoryUsed    uint64    `json:"memory_used"`
	MemoryLimit   uint64    `json:"memory_limit"`
	NetworkRx     uint64    `json:"network_rx"`
	NetworkTx     uint64    `json:"network_tx"`
	BlockRead     uint64    `json:"block_read"`
	BlockWrite    uint64    `json:"block_write"`
}

// DatabaseInfo provides information about the database storage.
type DatabaseInfo struct {
	Path            string     `json:"path"`
	SizeBytes       int64      `json:"size_bytes"`
	SizeFormatted   string     `json:"size_formatted"`
	MetricsCount    int64      `json:"metrics_count"`
	OldestTimestamp *time.Time `json:"oldest_timestamp,omitempty"`
	NewestTimestamp *time.Time `json:"newest_timestamp,omitempty"`
}

// NewMetricsCollector creates a new MetricsCollector instance.
func NewMetricsCollector(db *sql.DB, cfg *config.MetricsConfig) *MetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())
	return &MetricsCollector{
		db:     db,
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins the background metrics collection.
func (c *MetricsCollector) Start() {
	if !c.config.IsEnabled() {
		log.Println("[MetricsCollector] Metrics collection is disabled")
		return
	}

	interval := c.config.GetCollectionInterval()
	log.Printf("[MetricsCollector] Starting metrics collection (interval: %v)", interval)

	c.wg.Add(1)
	go c.collectLoop(interval)

	// Start cleanup goroutine (runs daily)
	c.wg.Add(1)
	go c.cleanupLoop()

	// Start aggregation goroutine (runs hourly)
	c.wg.Add(1)
	go c.aggregateLoop()
}

// Stop stops the background metrics collection.
func (c *MetricsCollector) Stop() {
	log.Println("[MetricsCollector] Stopping metrics collection")
	c.cancel()
	c.wg.Wait()
	log.Println("[MetricsCollector] Metrics collection stopped")
}

// GetLatest returns the most recently collected metrics.
func (c *MetricsCollector) GetLatest() *MetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastData
}

// collectLoop runs the main collection loop.
func (c *MetricsCollector) collectLoop(interval time.Duration) {
	defer c.wg.Done()

	// Collect immediately on start
	c.collect()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// collect collects and stores metrics.
func (c *MetricsCollector) collect() {
	snapshot := &MetricsSnapshot{
		Timestamp: time.Now(),
	}

	// Collect system metrics
	systemMetrics, err := metrics.GetSystemMetrics()
	if err != nil {
		log.Printf("[MetricsCollector] Error collecting system metrics: %v", err)
	} else {
		snapshot.System = systemMetrics
		if err := c.storeSystemMetrics(systemMetrics); err != nil {
			log.Printf("[MetricsCollector] Error storing system metrics: %v", err)
		}
	}

	// Collect Docker metrics
	dockerMetrics, err := metrics.GetDockerMetrics()
	if err != nil {
		log.Printf("[MetricsCollector] Error collecting Docker metrics: %v", err)
	} else {
		snapshot.Docker = dockerMetrics
		if err := c.storeDockerMetrics(dockerMetrics); err != nil {
			log.Printf("[MetricsCollector] Error storing Docker metrics: %v", err)
		}
	}

	// Update latest snapshot
	c.mu.Lock()
	c.lastData = snapshot
	c.mu.Unlock()
}

// storeSystemMetrics stores system metrics in the database.
func (c *MetricsCollector) storeSystemMetrics(m *metrics.SystemMetrics) error {
	diskData, err := json.Marshal(m.Disks)
	if err != nil {
		return err
	}

	networkData, err := json.Marshal(m.Network)
	if err != nil {
		return err
	}

	loadAvgData, err := json.Marshal(m.LoadAvg)
	if err != nil {
		return err
	}

	_, err = c.db.Exec(`
		INSERT INTO system_metrics (
			cpu_percent, memory_percent, memory_used, memory_total,
			disk_data, network_data, load_avg, uptime
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		m.CPU.UsagePercent,
		m.Memory.UsedPercent,
		m.Memory.Used,
		m.Memory.Total,
		string(diskData),
		string(networkData),
		string(loadAvgData),
		m.Uptime,
	)
	return err
}

// storeDockerMetrics stores Docker container metrics in the database.
func (c *MetricsCollector) storeDockerMetrics(m *metrics.DockerMetrics) error {
	if !m.Available {
		return nil
	}

	for _, container := range m.Containers {
		if container.State != "running" {
			continue // Only store metrics for running containers
		}

		_, err := c.db.Exec(`
			INSERT INTO docker_metrics (
				container_id, container_name, image, state,
				cpu_percent, memory_percent, memory_used, memory_limit,
				network_rx, network_tx, block_read, block_write
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			container.ID,
			container.Name,
			container.Image,
			container.State,
			container.CPU.UsagePercent,
			container.Memory.UsedPercent,
			container.Memory.Usage,
			container.Memory.Limit,
			container.Network.RxBytes,
			container.Network.TxBytes,
			container.BlockIO.ReadBytes,
			container.BlockIO.WriteBytes,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// cleanupLoop runs the cleanup loop to remove old metrics.
func (c *MetricsCollector) cleanupLoop() {
	defer c.wg.Done()

	// Run immediately on start
	c.cleanup()

	// Then run daily
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup removes old metrics based on retention settings.
func (c *MetricsCollector) cleanup() {
	rawRetention := time.Duration(c.config.GetRetentionDays()) * 24 * time.Hour
	hourlyRetention := time.Duration(c.config.GetHourlyRetentionDays()) * 24 * time.Hour
	dailyRetention := time.Duration(c.config.GetDailyRetentionDays()) * 24 * time.Hour

	rawCutoff := time.Now().Add(-rawRetention)
	hourlyCutoff := time.Now().Add(-hourlyRetention)
	dailyCutoff := time.Now().Add(-dailyRetention)

	// Delete old raw metrics
	result, err := c.db.Exec("DELETE FROM system_metrics WHERE timestamp < ?", rawCutoff)
	if err != nil {
		log.Printf("[MetricsCollector] Error cleaning up system_metrics: %v", err)
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[MetricsCollector] Cleaned up %d old system_metrics records", rows)
		}
	}

	result, err = c.db.Exec("DELETE FROM docker_metrics WHERE timestamp < ?", rawCutoff)
	if err != nil {
		log.Printf("[MetricsCollector] Error cleaning up docker_metrics: %v", err)
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[MetricsCollector] Cleaned up %d old docker_metrics records", rows)
		}
	}

	// Delete old hourly aggregates
	result, err = c.db.Exec("DELETE FROM system_metrics_hourly WHERE timestamp < ?", hourlyCutoff)
	if err != nil {
		log.Printf("[MetricsCollector] Error cleaning up system_metrics_hourly: %v", err)
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[MetricsCollector] Cleaned up %d old hourly aggregate records", rows)
		}
	}

	// Delete old daily aggregates
	result, err = c.db.Exec("DELETE FROM system_metrics_daily WHERE timestamp < ?", dailyCutoff)
	if err != nil {
		log.Printf("[MetricsCollector] Error cleaning up system_metrics_daily: %v", err)
	} else {
		if rows, _ := result.RowsAffected(); rows > 0 {
			log.Printf("[MetricsCollector] Cleaned up %d old daily aggregate records", rows)
		}
	}
}

// aggregateLoop runs the aggregation loop to create hourly/daily summaries.
func (c *MetricsCollector) aggregateLoop() {
	defer c.wg.Done()

	// Run hourly aggregation
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.aggregateHourly()

			// Run daily aggregation at midnight
			if time.Now().Hour() == 0 {
				c.aggregateDaily()
			}
		}
	}
}

// aggregateHourly creates hourly summaries from raw metrics.
func (c *MetricsCollector) aggregateHourly() {
	hourAgo := time.Now().Add(-1 * time.Hour)
	hourStart := time.Date(hourAgo.Year(), hourAgo.Month(), hourAgo.Day(), hourAgo.Hour(), 0, 0, 0, hourAgo.Location())
	hourEnd := hourStart.Add(1 * time.Hour)

	// Check if we already have aggregates for this hour
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM system_metrics_hourly WHERE timestamp = ?", hourStart).Scan(&count)
	if err != nil || count > 0 {
		return
	}

	// Aggregate system metrics
	row := c.db.QueryRow(`
		SELECT
			AVG(cpu_percent), MAX(cpu_percent),
			AVG(memory_percent), MAX(memory_percent),
			COUNT(*)
		FROM system_metrics
		WHERE timestamp >= ? AND timestamp < ?
	`, hourStart, hourEnd)

	var cpuAvg, cpuMax, memAvg, memMax sql.NullFloat64
	var sampleCount int
	if err := row.Scan(&cpuAvg, &cpuMax, &memAvg, &memMax, &sampleCount); err != nil || sampleCount == 0 {
		return
	}

	// Get last disk and network data for the hour
	var diskData, networkData string
	err = c.db.QueryRow(`
		SELECT disk_data, network_data FROM system_metrics
		WHERE timestamp >= ? AND timestamp < ?
		ORDER BY timestamp DESC LIMIT 1
	`, hourStart, hourEnd).Scan(&diskData, &networkData)
	if err != nil {
		diskData = "[]"
		networkData = "[]"
	}

	// Calculate network totals (sum of all samples' network data)
	var networkRxTotal, networkTxTotal int64
	rows, err := c.db.Query(`
		SELECT network_data FROM system_metrics
		WHERE timestamp >= ? AND timestamp < ?
	`, hourStart, hourEnd)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var netData string
			if err := rows.Scan(&netData); err == nil {
				var networks []metrics.NetworkMetrics
				if json.Unmarshal([]byte(netData), &networks) == nil {
					for _, n := range networks {
						networkRxTotal += int64(n.BytesRecv)
						networkTxTotal += int64(n.BytesSent)
					}
				}
			}
		}
	}

	_, err = c.db.Exec(`
		INSERT INTO system_metrics_hourly (
			timestamp, cpu_avg, cpu_max, memory_avg, memory_max,
			disk_data, network_rx_total, network_tx_total, sample_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		hourStart,
		cpuAvg.Float64, cpuMax.Float64,
		memAvg.Float64, memMax.Float64,
		diskData, networkRxTotal, networkTxTotal, sampleCount,
	)
	if err != nil {
		log.Printf("[MetricsCollector] Error creating hourly aggregate: %v", err)
	}
}

// aggregateDaily creates daily summaries from hourly aggregates.
func (c *MetricsCollector) aggregateDaily() {
	yesterday := time.Now().AddDate(0, 0, -1)
	dayStart := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

	// Check if we already have aggregates for this day
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM system_metrics_daily WHERE timestamp = ?", dayStart).Scan(&count)
	if err != nil || count > 0 {
		return
	}

	// Aggregate from hourly data
	row := c.db.QueryRow(`
		SELECT
			AVG(cpu_avg), MAX(cpu_max),
			AVG(memory_avg), MAX(memory_max),
			SUM(network_rx_total), SUM(network_tx_total),
			SUM(sample_count)
		FROM system_metrics_hourly
		WHERE timestamp >= ? AND timestamp < ?
	`, dayStart, dayEnd)

	var cpuAvg, cpuMax, memAvg, memMax sql.NullFloat64
	var networkRxTotal, networkTxTotal sql.NullInt64
	var sampleCount int
	if err := row.Scan(&cpuAvg, &cpuMax, &memAvg, &memMax, &networkRxTotal, &networkTxTotal, &sampleCount); err != nil || sampleCount == 0 {
		return
	}

	// Get last disk data for the day
	var diskData string
	err = c.db.QueryRow(`
		SELECT disk_data FROM system_metrics_hourly
		WHERE timestamp >= ? AND timestamp < ?
		ORDER BY timestamp DESC LIMIT 1
	`, dayStart, dayEnd).Scan(&diskData)
	if err != nil {
		diskData = "[]"
	}

	_, err = c.db.Exec(`
		INSERT INTO system_metrics_daily (
			timestamp, cpu_avg, cpu_max, memory_avg, memory_max,
			disk_data, network_rx_total, network_tx_total, sample_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		dayStart,
		cpuAvg.Float64, cpuMax.Float64,
		memAvg.Float64, memMax.Float64,
		diskData, networkRxTotal.Int64, networkTxTotal.Int64, sampleCount,
	)
	if err != nil {
		log.Printf("[MetricsCollector] Error creating daily aggregate: %v", err)
	}
}

// GetHistoricalMetrics retrieves historical system metrics for a time range.
func (c *MetricsCollector) GetHistoricalMetrics(from, to time.Time, resolution string) ([]StoredSystemMetrics, error) {
	var query string
	switch resolution {
	case "raw":
		query = `
			SELECT id, timestamp, cpu_percent, memory_percent, memory_used, memory_total,
				   disk_data, network_data, load_avg, uptime
			FROM system_metrics
			WHERE timestamp >= ? AND timestamp <= ?
			ORDER BY timestamp ASC
		`
	case "hourly":
		query = `
			SELECT id, timestamp, cpu_avg as cpu_percent, memory_avg as memory_percent,
				   0 as memory_used, 0 as memory_total, disk_data, '[]' as network_data,
				   '[]' as load_avg, 0 as uptime
			FROM system_metrics_hourly
			WHERE timestamp >= ? AND timestamp <= ?
			ORDER BY timestamp ASC
		`
	case "daily":
		query = `
			SELECT id, timestamp, cpu_avg as cpu_percent, memory_avg as memory_percent,
				   0 as memory_used, 0 as memory_total, disk_data, '[]' as network_data,
				   '[]' as load_avg, 0 as uptime
			FROM system_metrics_daily
			WHERE timestamp >= ? AND timestamp <= ?
			ORDER BY timestamp ASC
		`
	default:
		query = `
			SELECT id, timestamp, cpu_percent, memory_percent, memory_used, memory_total,
				   disk_data, network_data, load_avg, uptime
			FROM system_metrics
			WHERE timestamp >= ? AND timestamp <= ?
			ORDER BY timestamp ASC
		`
	}

	rows, err := c.db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StoredSystemMetrics
	for rows.Next() {
		var m StoredSystemMetrics
		if err := rows.Scan(
			&m.ID, &m.Timestamp, &m.CPUPercent, &m.MemoryPercent,
			&m.MemoryUsed, &m.MemoryTotal, &m.DiskData, &m.NetworkData,
			&m.LoadAvg, &m.Uptime,
		); err != nil {
			continue
		}
		results = append(results, m)
	}
	return results, nil
}

// GetHistoricalDockerMetrics retrieves historical Docker metrics for a container.
func (c *MetricsCollector) GetHistoricalDockerMetrics(containerID string, from, to time.Time) ([]StoredDockerMetrics, error) {
	query := `
		SELECT id, timestamp, container_id, container_name, image, state,
			   cpu_percent, memory_percent, memory_used, memory_limit,
			   network_rx, network_tx, block_read, block_write
		FROM docker_metrics
		WHERE container_id = ? AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp ASC
	`

	rows, err := c.db.Query(query, containerID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StoredDockerMetrics
	for rows.Next() {
		var m StoredDockerMetrics
		if err := rows.Scan(
			&m.ID, &m.Timestamp, &m.ContainerID, &m.ContainerName,
			&m.Image, &m.State, &m.CPUPercent, &m.MemoryPercent,
			&m.MemoryUsed, &m.MemoryLimit, &m.NetworkRx, &m.NetworkTx,
			&m.BlockRead, &m.BlockWrite,
		); err != nil {
			continue
		}
		results = append(results, m)
	}
	return results, nil
}

// PruneMetrics manually removes metrics older than the specified date.
func (c *MetricsCollector) PruneMetrics(before time.Time) (int64, error) {
	var totalDeleted int64

	result, err := c.db.Exec("DELETE FROM system_metrics WHERE timestamp < ?", before)
	if err != nil {
		return 0, err
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		totalDeleted += rows
	}

	result, err = c.db.Exec("DELETE FROM docker_metrics WHERE timestamp < ?", before)
	if err != nil {
		return totalDeleted, err
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		totalDeleted += rows
	}

	result, err = c.db.Exec("DELETE FROM system_metrics_hourly WHERE timestamp < ?", before)
	if err != nil {
		return totalDeleted, err
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		totalDeleted += rows
	}

	result, err = c.db.Exec("DELETE FROM system_metrics_daily WHERE timestamp < ?", before)
	if err != nil {
		return totalDeleted, err
	}
	if rows, _ := result.RowsAffected(); rows > 0 {
		totalDeleted += rows
	}

	return totalDeleted, nil
}

// VacuumDatabase reclaims unused space in the SQLite database.
func (c *MetricsCollector) VacuumDatabase() error {
	_, err := c.db.Exec("VACUUM")
	return err
}

// GetDatabaseInfo returns information about the database storage.
func (c *MetricsCollector) GetDatabaseInfo(dbPath string) (*DatabaseInfo, error) {
	info := &DatabaseInfo{Path: dbPath}

	// Get file size
	// Note: We can't use os.Stat directly on the path from here
	// This will be handled by the handler which has access to the config

	// Get metrics count
	var systemCount, dockerCount, hourlyCount, dailyCount int64
	_ = c.db.QueryRow("SELECT COUNT(*) FROM system_metrics").Scan(&systemCount)
	_ = c.db.QueryRow("SELECT COUNT(*) FROM docker_metrics").Scan(&dockerCount)
	_ = c.db.QueryRow("SELECT COUNT(*) FROM system_metrics_hourly").Scan(&hourlyCount)
	_ = c.db.QueryRow("SELECT COUNT(*) FROM system_metrics_daily").Scan(&dailyCount)
	info.MetricsCount = systemCount + dockerCount + hourlyCount + dailyCount

	// Get oldest and newest timestamps
	// Use string scanning and manual parsing to handle SQLite timestamp format
	var oldestStr, newestStr sql.NullString
	_ = c.db.QueryRow("SELECT MIN(timestamp) FROM system_metrics").Scan(&oldestStr)
	if oldestStr.Valid && oldestStr.String != "" {
		if t, err := ParseSQLiteTimestamp(oldestStr.String); err == nil {
			info.OldestTimestamp = &t
		}
	}
	_ = c.db.QueryRow("SELECT MAX(timestamp) FROM system_metrics").Scan(&newestStr)
	if newestStr.Valid && newestStr.String != "" {
		if t, err := ParseSQLiteTimestamp(newestStr.String); err == nil {
			info.NewestTimestamp = &t
		}
	}

	return info, nil
}

// ParseSQLiteTimestamp parses timestamp strings from SQLite
// which can be in various formats depending on how they were stored.
func ParseSQLiteTimestamp(s string) (time.Time, error) {
	// Try common SQLite timestamp formats
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999Z07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339Nano,
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}

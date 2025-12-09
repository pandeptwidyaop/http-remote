// Package handlers provides HTTP request handlers for the web UI and API.
package handlers

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/metrics"
	"github.com/pandeptwidyaop/http-remote/internal/services"
)

// MetricsHandler handles HTTP requests for system and Docker metrics.
type MetricsHandler struct {
	collector *services.MetricsCollector
	dbPath    string
}

// NewMetricsHandler creates a new MetricsHandler instance.
func NewMetricsHandler(collector *services.MetricsCollector, dbPath string) *MetricsHandler {
	return &MetricsHandler{
		collector: collector,
		dbPath:    dbPath,
	}
}

// MetricsSummary represents a quick overview of system and Docker status.
type MetricsSummary struct {
	System struct {
		CPUPercent    float64 `json:"cpu_percent"`
		MemoryPercent float64 `json:"memory_percent"`
		DiskCount     int     `json:"disk_count"`
		NetworkCount  int     `json:"network_count"`
		Uptime        int64   `json:"uptime"`
	} `json:"system"`
	Docker struct {
		Available bool `json:"available"`
		Running   int  `json:"running"`
		Stopped   int  `json:"stopped"`
		Total     int  `json:"total"`
	} `json:"docker"`
}

// GetSystem returns current system metrics.
// GET /api/metrics/system
func (h *MetricsHandler) GetSystem(c *gin.Context) {
	systemMetrics, err := metrics.GetSystemMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, systemMetrics)
}

// GetDocker returns all Docker container metrics.
// GET /api/metrics/docker
func (h *MetricsHandler) GetDocker(c *gin.Context) {
	dockerMetrics, err := metrics.GetDockerMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dockerMetrics)
}

// GetContainer returns metrics for a specific Docker container.
// GET /api/metrics/docker/:id
func (h *MetricsHandler) GetContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	containerMetrics, err := metrics.GetContainerMetrics(containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, containerMetrics)
}

// GetSummary returns a quick summary of system and Docker metrics.
// GET /api/metrics/summary
func (h *MetricsHandler) GetSummary(c *gin.Context) {
	summary := MetricsSummary{}

	// Get system metrics
	systemMetrics, err := metrics.GetSystemMetrics()
	if err == nil {
		summary.System.CPUPercent = systemMetrics.CPU.UsagePercent
		summary.System.MemoryPercent = systemMetrics.Memory.UsedPercent
		summary.System.DiskCount = len(systemMetrics.Disks)
		summary.System.NetworkCount = len(systemMetrics.Network)
		summary.System.Uptime = systemMetrics.Uptime
	}

	// Get Docker metrics
	dockerMetrics, err := metrics.GetDockerMetrics()
	if err == nil {
		summary.Docker.Available = dockerMetrics.Available
		summary.Docker.Running = dockerMetrics.Summary.Running
		summary.Docker.Stopped = dockerMetrics.Summary.Stopped
		summary.Docker.Total = dockerMetrics.Summary.Total
	}

	c.JSON(http.StatusOK, summary)
}

// GetHistorical returns historical system metrics for a time range.
// GET /api/metrics/history?from=<timestamp>&to=<timestamp>&resolution=<raw|hourly|daily>
func (h *MetricsHandler) GetHistorical(c *gin.Context) {
	fromStr := c.DefaultQuery("from", "")
	toStr := c.DefaultQuery("to", "")
	resolution := c.DefaultQuery("resolution", "raw")

	var from, to time.Time
	var err error

	if fromStr == "" {
		from = time.Now().Add(-24 * time.Hour) // Default: last 24 hours
	} else {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' timestamp format, use RFC3339"})
			return
		}
	}

	if toStr == "" {
		to = time.Now()
	} else {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' timestamp format, use RFC3339"})
			return
		}
	}

	if h.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collector not available"})
		return
	}

	data, err := h.collector.GetHistoricalMetrics(from, to, resolution)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"from":       from,
		"to":         to,
		"resolution": resolution,
		"data":       data,
	})
}

// GetContainerHistory returns historical Docker metrics for a specific container.
// GET /api/metrics/docker/:id/history?from=<timestamp>&to=<timestamp>
func (h *MetricsHandler) GetContainerHistory(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID required"})
		return
	}

	fromStr := c.DefaultQuery("from", "")
	toStr := c.DefaultQuery("to", "")

	var from, to time.Time
	var err error

	if fromStr == "" {
		from = time.Now().Add(-24 * time.Hour)
	} else {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'from' timestamp format, use RFC3339"})
			return
		}
	}

	if toStr == "" {
		to = time.Now()
	} else {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid 'to' timestamp format, use RFC3339"})
			return
		}
	}

	if h.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collector not available"})
		return
	}

	data, err := h.collector.GetHistoricalDockerMetrics(containerID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"container_id": containerID,
		"from":         from,
		"to":           to,
		"data":         data,
	})
}

// GetDatabaseInfo returns information about the metrics database storage.
// GET /api/metrics/storage
func (h *MetricsHandler) GetDatabaseInfo(c *gin.Context) {
	if h.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collector not available"})
		return
	}

	info, err := h.collector.GetDatabaseInfo(h.dbPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get file size
	if stat, err := os.Stat(h.dbPath); err == nil {
		info.SizeBytes = stat.Size()
		info.SizeFormatted = formatBytes(info.SizeBytes)
	}

	c.JSON(http.StatusOK, info)
}

// PruneMetrics manually removes metrics older than the specified date.
// POST /api/metrics/prune
func (h *MetricsHandler) PruneMetrics(c *gin.Context) {
	var req struct {
		Before string `json:"before" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "before timestamp is required"})
		return
	}

	before, err := time.Parse(time.RFC3339, req.Before)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp format, use RFC3339"})
		return
	}

	if h.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collector not available"})
		return
	}

	deleted, err := h.collector.PruneMetrics(before)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"deleted_records": deleted,
		"pruned_before":   before,
	})
}

// VacuumDatabase reclaims unused space in the database.
// POST /api/metrics/vacuum
func (h *MetricsHandler) VacuumDatabase(c *gin.Context) {
	if h.collector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics collector not available"})
		return
	}

	// Get size before
	var sizeBefore int64
	if stat, err := os.Stat(h.dbPath); err == nil {
		sizeBefore = stat.Size()
	}

	if err := h.collector.VacuumDatabase(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get size after
	var sizeAfter int64
	if stat, err := os.Stat(h.dbPath); err == nil {
		sizeAfter = stat.Size()
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"size_before":     sizeBefore,
		"size_after":      sizeAfter,
		"space_reclaimed": sizeBefore - sizeAfter,
	})
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

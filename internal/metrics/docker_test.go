package metrics

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
)

func TestGetDockerMetrics(t *testing.T) {
	metrics, err := GetDockerMetrics()
	if err != nil {
		t.Fatalf("GetDockerMetrics returned error: %v", err)
	}

	if metrics == nil {
		t.Fatal("GetDockerMetrics returned nil metrics")
	}

	// The function should always return a DockerMetrics struct
	// Available will be false if Docker is not running
	if !metrics.Available {
		t.Log("Docker is not available - skipping container-specific tests")
		return
	}

	t.Run("Docker version present", func(t *testing.T) {
		if metrics.Version == "" {
			t.Error("Docker version should not be empty when Docker is available")
		}
	})

	t.Run("Summary matches containers", func(t *testing.T) {
		running := 0
		paused := 0
		stopped := 0

		for _, c := range metrics.Containers {
			switch c.State {
			case "running":
				running++
			case "paused":
				paused++
			default:
				stopped++
			}
		}

		if metrics.Summary.Running != running {
			t.Errorf("Summary running count mismatch: expected %d, got %d", running, metrics.Summary.Running)
		}
		if metrics.Summary.Paused != paused {
			t.Errorf("Summary paused count mismatch: expected %d, got %d", paused, metrics.Summary.Paused)
		}
		if metrics.Summary.Stopped != stopped {
			t.Errorf("Summary stopped count mismatch: expected %d, got %d", stopped, metrics.Summary.Stopped)
		}
		if metrics.Summary.Total != len(metrics.Containers) {
			t.Errorf("Summary total count mismatch: expected %d, got %d", len(metrics.Containers), metrics.Summary.Total)
		}
	})

	t.Run("Container IDs are truncated", func(t *testing.T) {
		for _, c := range metrics.Containers {
			if len(c.ID) > 12 {
				t.Errorf("Container ID should be truncated to 12 chars, got %d", len(c.ID))
			}
		}
	})
}

func TestGetDockerMetrics_UnavailableDocker(t *testing.T) {
	// This test verifies that we gracefully handle Docker being unavailable
	// The function should return a struct with Available=false, not an error
	metrics, err := GetDockerMetrics()
	if err != nil {
		t.Fatalf("GetDockerMetrics should not return error even if Docker is unavailable: %v", err)
	}

	if metrics == nil {
		t.Fatal("GetDockerMetrics should return a valid struct even if Docker is unavailable")
	}

	// If Docker is not available, verify the struct is properly initialized
	if !metrics.Available {
		if metrics.Containers == nil {
			t.Error("Containers should be initialized to empty slice, not nil")
		}
	}
}

func TestIsDockerAvailable(t *testing.T) {
	// This test just ensures the function doesn't panic
	result := IsDockerAvailable()
	t.Logf("Docker available: %v", result)
	// We can't assert a specific value as it depends on the environment
}

func TestCalculateCPUPercent(t *testing.T) {
	tests := []struct {
		name     string
		stats    *types.StatsJSON
		expected float64
	}{
		{
			name: "Zero deltas",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 1000,
						},
						SystemUsage: 2000,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 1000, // Same as current
						},
						SystemUsage: 2000, // Same as current
					},
				},
			},
			expected: 0.0,
		},
		{
			name: "Normal CPU usage",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 2000,
						},
						SystemUsage: 10000,
						OnlineCPUs:  4,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 1000,
						},
						SystemUsage: 5000,
					},
				},
			},
			expected: 80.0, // (1000/5000) * 4 * 100 = 80%
		},
		{
			name: "No online CPUs (fallback to per-cpu usage)",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage:  2000,
							PercpuUsage: []uint64{500, 500, 500, 500}, // 4 CPUs
						},
						SystemUsage: 10000,
						OnlineCPUs:  0, // Not set
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 1000,
						},
						SystemUsage: 5000,
					},
				},
			},
			expected: 80.0, // (1000/5000) * 4 * 100 = 80%
		},
		{
			name: "Single CPU",
			stats: &types.StatsJSON{
				Stats: types.Stats{
					CPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 2000,
						},
						SystemUsage: 10000,
						OnlineCPUs:  1,
					},
					PreCPUStats: types.CPUStats{
						CPUUsage: types.CPUUsage{
							TotalUsage: 1000,
						},
						SystemUsage: 5000,
					},
				},
			},
			expected: 20.0, // (1000/5000) * 1 * 100 = 20%
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateCPUPercent(tc.stats)
			// Allow small floating point tolerance
			tolerance := 0.01
			if result < tc.expected-tolerance || result > tc.expected+tolerance {
				t.Errorf("calculateCPUPercent() = %v, expected %v", result, tc.expected)
			}
		})
	}
}

func TestDockerMetricsStructure(t *testing.T) {
	// Test that the structs can be properly initialized
	metrics := &DockerMetrics{
		Available: true,
		Version:   "20.10.17",
		Containers: []ContainerMetrics{
			{
				ID:      "abc123def456",
				Name:    "test-container",
				Image:   "nginx:latest",
				Status:  "Up 2 hours",
				State:   "running",
				Created: time.Now(),
				CPU: ContainerCPU{
					UsagePercent: 25.5,
				},
				Memory: ContainerMemory{
					Usage:       1073741824, // 1GB
					Limit:       4294967296, // 4GB
					UsedPercent: 25.0,
					Cache:       268435456, // 256MB
				},
				Network: ContainerNetwork{
					RxBytes:   10485760, // 10MB
					TxBytes:   5242880,  // 5MB
					RxPackets: 10000,
					TxPackets: 5000,
				},
				BlockIO: ContainerBlockIO{
					ReadBytes:  52428800, // 50MB
					WriteBytes: 26214400, // 25MB
				},
			},
		},
		Summary: DockerSummary{
			Total:   5,
			Running: 3,
			Paused:  0,
			Stopped: 2,
		},
	}

	if metrics.Version != "20.10.17" {
		t.Error("Version not set correctly")
	}
	if len(metrics.Containers) != 1 {
		t.Error("Containers not set correctly")
	}
	if metrics.Summary.Total != 5 {
		t.Error("Summary total not set correctly")
	}

	container := metrics.Containers[0]
	if container.Name != "test-container" {
		t.Error("Container name not set correctly")
	}
	if container.CPU.UsagePercent != 25.5 {
		t.Error("Container CPU usage not set correctly")
	}
	if container.Memory.UsedPercent != 25.0 {
		t.Error("Container memory percent not set correctly")
	}
}

func TestContainerMetricsFields(t *testing.T) {
	// Test ContainerMetrics structure with all fields
	container := ContainerMetrics{
		ID:      "deadbeef1234",
		Name:    "my-app",
		Image:   "myapp:v1.0",
		Status:  "Up 5 minutes",
		State:   "running",
		Created: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	if container.ID != "deadbeef1234" {
		t.Errorf("ID = %s, want deadbeef1234", container.ID)
	}
	if container.Name != "my-app" {
		t.Errorf("Name = %s, want my-app", container.Name)
	}
	if container.Image != "myapp:v1.0" {
		t.Errorf("Image = %s, want myapp:v1.0", container.Image)
	}
	if container.State != "running" {
		t.Errorf("State = %s, want running", container.State)
	}
}

func TestDockerSummaryStates(t *testing.T) {
	tests := []struct {
		name    string
		summary DockerSummary
	}{
		{
			name: "All running",
			summary: DockerSummary{
				Total:   5,
				Running: 5,
				Paused:  0,
				Stopped: 0,
			},
		},
		{
			name: "All stopped",
			summary: DockerSummary{
				Total:   5,
				Running: 0,
				Paused:  0,
				Stopped: 5,
			},
		},
		{
			name: "Mixed states",
			summary: DockerSummary{
				Total:   10,
				Running: 5,
				Paused:  2,
				Stopped: 3,
			},
		},
		{
			name: "Empty",
			summary: DockerSummary{
				Total:   0,
				Running: 0,
				Paused:  0,
				Stopped: 0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Verify sum equals total
			sum := tc.summary.Running + tc.summary.Paused + tc.summary.Stopped
			if sum != tc.summary.Total {
				t.Errorf("Sum of states (%d) != Total (%d)", sum, tc.summary.Total)
			}
		})
	}
}

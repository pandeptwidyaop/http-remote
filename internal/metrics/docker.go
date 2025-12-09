package metrics

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// DockerMetrics represents all Docker container metrics.
type DockerMetrics struct {
	Available  bool               `json:"available"`
	Version    string             `json:"version,omitempty"`
	Containers []ContainerMetrics `json:"containers"`
	Summary    DockerSummary      `json:"summary"`
}

// DockerSummary provides a quick overview of container states.
type DockerSummary struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Paused  int `json:"paused"`
	Stopped int `json:"stopped"`
}

// ContainerMetrics represents metrics for a single container.
type ContainerMetrics struct {
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	Image   string           `json:"image"`
	Status  string           `json:"status"`
	State   string           `json:"state"`
	Created time.Time        `json:"created"`
	CPU     ContainerCPU     `json:"cpu"`
	Memory  ContainerMemory  `json:"memory"`
	Network ContainerNetwork `json:"network"`
	BlockIO ContainerBlockIO `json:"block_io"`
}

// ContainerCPU represents CPU usage for a container.
type ContainerCPU struct {
	UsagePercent float64 `json:"usage_percent"`
}

// ContainerMemory represents memory usage for a container.
type ContainerMemory struct {
	Usage       uint64  `json:"usage"`
	Limit       uint64  `json:"limit"`
	UsedPercent float64 `json:"used_percent"`
	Cache       uint64  `json:"cache"`
}

// ContainerNetwork represents network I/O for a container.
type ContainerNetwork struct {
	RxBytes   uint64 `json:"rx_bytes"`
	TxBytes   uint64 `json:"tx_bytes"`
	RxPackets uint64 `json:"rx_packets"`
	TxPackets uint64 `json:"tx_packets"`
}

// ContainerBlockIO represents block I/O for a container.
type ContainerBlockIO struct {
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
}

// GetDockerMetrics collects metrics for all Docker containers.
func GetDockerMetrics() (*DockerMetrics, error) {
	return GetDockerMetricsWithContext(context.Background())
}

// GetDockerMetricsWithContext collects Docker metrics with context cancellation support.
func GetDockerMetricsWithContext(parentCtx context.Context) (*DockerMetrics, error) {
	// Check if already canceled
	if parentCtx.Err() != nil {
		return nil, parentCtx.Err()
	}

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return &DockerMetrics{Available: false}, nil
	}
	defer func() { _ = cli.Close() }()

	// Check Docker is available
	info, err := cli.Info(ctx)
	if err != nil {
		return &DockerMetrics{Available: false}, nil
	}

	metrics := &DockerMetrics{
		Available:  true,
		Version:    info.ServerVersion,
		Containers: make([]ContainerMetrics, 0),
	}

	// List all containers
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return metrics, nil
	}

	metrics.Summary.Total = len(containers)

	// Count summary first (fast)
	for _, c := range containers {
		switch c.State {
		case "running":
			metrics.Summary.Running++
		case "paused":
			metrics.Summary.Paused++
		default:
			metrics.Summary.Stopped++
		}
	}

	// Collect container stats in parallel for running containers
	var wg sync.WaitGroup
	var mu sync.Mutex
	containerMetrics := make([]ContainerMetrics, len(containers))

	for i, c := range containers {
		containerMetrics[i] = ContainerMetrics{
			ID:      c.ID[:12],
			Name:    strings.TrimPrefix(c.Names[0], "/"),
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Created: time.Unix(c.Created, 0),
		}

		// Only get stats for running containers - in parallel
		if c.State == "running" {
			wg.Add(1)
			go func(idx int, containerID string) {
				defer wg.Done()

				stats, err := cli.ContainerStats(ctx, containerID, false)
				if err != nil {
					return
				}
				defer func() { _ = stats.Body.Close() }()

				var statsJSON types.StatsJSON
				if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err != nil {
					return
				}

				mu.Lock()
				defer mu.Unlock()

				// Calculate CPU percentage
				containerMetrics[idx].CPU.UsagePercent = calculateCPUPercent(&statsJSON)

				// Memory
				if statsJSON.MemoryStats.Limit > 0 {
					containerMetrics[idx].Memory = ContainerMemory{
						Usage:       statsJSON.MemoryStats.Usage,
						Limit:       statsJSON.MemoryStats.Limit,
						UsedPercent: float64(statsJSON.MemoryStats.Usage) / float64(statsJSON.MemoryStats.Limit) * 100,
						Cache:       statsJSON.MemoryStats.Stats["cache"],
					}
				}

				// Network stats (sum all interfaces)
				for _, netStats := range statsJSON.Networks {
					containerMetrics[idx].Network.RxBytes += netStats.RxBytes
					containerMetrics[idx].Network.TxBytes += netStats.TxBytes
					containerMetrics[idx].Network.RxPackets += netStats.RxPackets
					containerMetrics[idx].Network.TxPackets += netStats.TxPackets
				}

				// Block I/O
				for _, bio := range statsJSON.BlkioStats.IoServiceBytesRecursive {
					switch bio.Op {
					case "read", "Read":
						containerMetrics[idx].BlockIO.ReadBytes += bio.Value
					case "write", "Write":
						containerMetrics[idx].BlockIO.WriteBytes += bio.Value
					}
				}
			}(i, c.ID)
		}
	}

	wg.Wait()
	metrics.Containers = containerMetrics

	return metrics, nil
}

// GetContainerMetrics collects metrics for a specific container.
func GetContainerMetrics(containerID string) (*ContainerMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer func() { _ = cli.Close() }()

	// Get container info
	containerInfo, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Parse created time from string
	createdTime, _ := time.Parse(time.RFC3339Nano, containerInfo.Created)

	containerMetric := &ContainerMetrics{
		ID:      containerInfo.ID[:12],
		Name:    strings.TrimPrefix(containerInfo.Name, "/"),
		Image:   containerInfo.Config.Image,
		Status:  containerInfo.State.Status,
		State:   containerInfo.State.Status,
		Created: createdTime,
	}

	// Only get stats for running containers
	if containerInfo.State.Running {
		stats, err := cli.ContainerStats(ctx, containerID, false)
		if err == nil {
			var statsJSON types.StatsJSON
			if err := json.NewDecoder(stats.Body).Decode(&statsJSON); err == nil {
				containerMetric.CPU.UsagePercent = calculateCPUPercent(&statsJSON)

				if statsJSON.MemoryStats.Limit > 0 {
					containerMetric.Memory = ContainerMemory{
						Usage:       statsJSON.MemoryStats.Usage,
						Limit:       statsJSON.MemoryStats.Limit,
						UsedPercent: float64(statsJSON.MemoryStats.Usage) / float64(statsJSON.MemoryStats.Limit) * 100,
						Cache:       statsJSON.MemoryStats.Stats["cache"],
					}
				}

				for _, netStats := range statsJSON.Networks {
					containerMetric.Network.RxBytes += netStats.RxBytes
					containerMetric.Network.TxBytes += netStats.TxBytes
					containerMetric.Network.RxPackets += netStats.RxPackets
					containerMetric.Network.TxPackets += netStats.TxPackets
				}

				for _, bio := range statsJSON.BlkioStats.IoServiceBytesRecursive {
					switch bio.Op {
					case "read", "Read":
						containerMetric.BlockIO.ReadBytes += bio.Value
					case "write", "Write":
						containerMetric.BlockIO.WriteBytes += bio.Value
					}
				}
			}
			_ = stats.Body.Close()
		}
	}

	return containerMetric, nil
}

// calculateCPUPercent calculates CPU usage percentage from Docker stats.
func calculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0 && cpuDelta > 0 {
		cpuCount := float64(stats.CPUStats.OnlineCPUs)
		if cpuCount == 0 {
			cpuCount = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		if cpuCount == 0 {
			cpuCount = 1
		}
		return (cpuDelta / systemDelta) * cpuCount * 100.0
	}
	return 0.0
}

// IsDockerAvailable checks if Docker daemon is accessible.
func IsDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false
	}
	defer func() { _ = cli.Close() }()

	_, err = cli.Ping(ctx)
	return err == nil
}

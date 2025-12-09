// Package metrics provides system and Docker metrics collection.
package metrics

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemMetrics represents current system resource usage.
type SystemMetrics struct {
	CPU     CPUMetrics       `json:"cpu"`
	Memory  MemoryMetrics    `json:"memory"`
	Disks   []DiskMetrics    `json:"disks"`
	Network []NetworkMetrics `json:"network"`
	Uptime  int64            `json:"uptime"`   // seconds
	LoadAvg []float64        `json:"load_avg"` // 1, 5, 15 min
}

// CPUMetrics represents CPU usage information.
type CPUMetrics struct {
	UsagePercent float64   `json:"usage_percent"`
	Cores        int       `json:"cores"`
	PerCore      []float64 `json:"per_core"`
}

// MemoryMetrics represents memory usage information.
type MemoryMetrics struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
}

// DiskMetrics represents disk usage information.
type DiskMetrics struct {
	Device      string  `json:"device"`
	MountPoint  string  `json:"mount_point"`
	Filesystem  string  `json:"filesystem"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkMetrics represents network interface statistics.
type NetworkMetrics struct {
	Interface   string `json:"interface"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrIn       uint64 `json:"err_in"`
	ErrOut      uint64 `json:"err_out"`
	DropIn      uint64 `json:"drop_in"`
	DropOut     uint64 `json:"drop_out"`
	IsUp        bool   `json:"is_up"`
}

// GetSystemMetrics collects current system metrics using parallel goroutines for speed.
func GetSystemMetrics() (*SystemMetrics, error) {
	return GetSystemMetricsWithContext(context.Background())
}

// GetSystemMetricsWithContext collects system metrics with context cancellation support.
func GetSystemMetricsWithContext(ctx context.Context) (*SystemMetrics, error) {
	// Check if already canceled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	metrics := &SystemMetrics{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// CPU metrics (slowest - runs in parallel)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Check context before slow operation
		if ctx.Err() != nil {
			return
		}
		// Single call with 200ms interval (reduced from 500ms)
		cpuPercent, err := cpu.Percent(200*time.Millisecond, true)
		if err == nil && len(cpuPercent) > 0 {
			mu.Lock()
			metrics.CPU.PerCore = cpuPercent
			// Calculate average
			var total float64
			for _, p := range cpuPercent {
				total += p
			}
			metrics.CPU.UsagePercent = total / float64(len(cpuPercent))
			mu.Unlock()
		}

		if ctx.Err() != nil {
			return
		}
		cpuCount, err := cpu.Counts(true)
		if err == nil {
			mu.Lock()
			metrics.CPU.Cores = cpuCount
			mu.Unlock()
		}
	}()

	// Memory metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ctx.Err() != nil {
			return
		}
		vmem, err := mem.VirtualMemory()
		if err == nil {
			mu.Lock()
			metrics.Memory = MemoryMetrics{
				Total:       vmem.Total,
				Used:        vmem.Used,
				Available:   vmem.Available,
				UsedPercent: vmem.UsedPercent,
			}
			mu.Unlock()
		}

		if ctx.Err() != nil {
			return
		}
		swap, err := mem.SwapMemory()
		if err == nil {
			mu.Lock()
			metrics.Memory.SwapTotal = swap.Total
			metrics.Memory.SwapUsed = swap.Used
			mu.Unlock()
		}
	}()

	// Disk metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ctx.Err() != nil {
			return
		}
		partitions, err := disk.Partitions(false)
		if err != nil {
			return
		}

		var disks []DiskMetrics
		for _, p := range partitions {
			if ctx.Err() != nil {
				return
			}
			if isVirtualFilesystem(p.Fstype) {
				continue
			}

			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}

			disks = append(disks, DiskMetrics{
				Device:      p.Device,
				MountPoint:  p.Mountpoint,
				Filesystem:  p.Fstype,
				Total:       usage.Total,
				Used:        usage.Used,
				Available:   usage.Free,
				UsedPercent: usage.UsedPercent,
			})
		}

		mu.Lock()
		metrics.Disks = disks
		mu.Unlock()
	}()

	// Network metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ctx.Err() != nil {
			return
		}
		netIO, err := net.IOCounters(true)
		if err != nil {
			return
		}

		netInterfaces, _ := net.Interfaces()
		ifaceFlags := make(map[string]bool)
		for _, iface := range netInterfaces {
			isUp := false
			for _, flag := range iface.Flags {
				if flag == "up" {
					isUp = true
					break
				}
			}
			ifaceFlags[iface.Name] = isUp
		}

		var networks []NetworkMetrics
		for _, io := range netIO {
			if isVirtualInterface(io.Name) {
				continue
			}

			networks = append(networks, NetworkMetrics{
				Interface:   io.Name,
				BytesSent:   io.BytesSent,
				BytesRecv:   io.BytesRecv,
				PacketsSent: io.PacketsSent,
				PacketsRecv: io.PacketsRecv,
				ErrIn:       io.Errin,
				ErrOut:      io.Errout,
				DropIn:      io.Dropin,
				DropOut:     io.Dropout,
				IsUp:        ifaceFlags[io.Name],
			})
		}

		mu.Lock()
		metrics.Network = networks
		mu.Unlock()
	}()

	// Host info (uptime & load)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if ctx.Err() != nil {
			return
		}
		hostInfo, err := host.Info()
		if err == nil {
			mu.Lock()
			metrics.Uptime = int64(hostInfo.Uptime)
			mu.Unlock()
		}

		if ctx.Err() != nil {
			return
		}
		loadAvg, err := load.Avg()
		if err == nil {
			mu.Lock()
			metrics.LoadAvg = []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
			mu.Unlock()
		}
	}()

	// Wait for all goroutines to complete
	wg.Wait()

	return metrics, nil
}

// isVirtualFilesystem returns true if the filesystem type is virtual.
func isVirtualFilesystem(fstype string) bool {
	virtualFS := []string{
		"sysfs", "proc", "devfs", "devpts", "tmpfs", "debugfs",
		"securityfs", "cgroup", "cgroup2", "pstore", "bpf",
		"autofs", "mqueue", "hugetlbfs", "fusectl", "configfs",
		"devtmpfs", "overlay", "squashfs", "nsfs", "ramfs",
	}

	for _, vfs := range virtualFS {
		if fstype == vfs {
			return true
		}
	}
	return false
}

// isVirtualInterface returns true if the network interface is virtual.
func isVirtualInterface(name string) bool {
	// Skip loopback
	if name == "lo" || name == "lo0" {
		return true
	}

	// Skip Docker/container virtual interfaces
	virtualPrefixes := []string{
		"veth", "docker", "br-", "virbr", "vnet",
		"flannel", "cni", "calico", "weave",
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

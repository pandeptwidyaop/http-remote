// Package metrics provides system and Docker metrics collection.
package metrics

import (
	"strings"
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

// GetSystemMetrics collects current system metrics.
func GetSystemMetrics() (*SystemMetrics, error) {
	metrics := &SystemMetrics{}

	// CPU - use a shorter interval for faster response
	cpuPercent, err := cpu.Percent(500*time.Millisecond, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics.CPU.UsagePercent = cpuPercent[0]
	}

	cpuPerCore, err := cpu.Percent(500*time.Millisecond, true)
	if err == nil {
		metrics.CPU.PerCore = cpuPerCore
	}

	cpuCount, err := cpu.Counts(true)
	if err == nil {
		metrics.CPU.Cores = cpuCount
	}

	// Memory
	vmem, err := mem.VirtualMemory()
	if err == nil {
		metrics.Memory = MemoryMetrics{
			Total:       vmem.Total,
			Used:        vmem.Used,
			Available:   vmem.Available,
			UsedPercent: vmem.UsedPercent,
		}
	}

	swap, err := mem.SwapMemory()
	if err == nil {
		metrics.Memory.SwapTotal = swap.Total
		metrics.Memory.SwapUsed = swap.Used
	}

	// Disks - all partitions
	partitions, err := disk.Partitions(false)
	if err == nil {
		for _, p := range partitions {
			// Skip special filesystems
			if isVirtualFilesystem(p.Fstype) {
				continue
			}

			usage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				continue
			}

			metrics.Disks = append(metrics.Disks, DiskMetrics{
				Device:      p.Device,
				MountPoint:  p.Mountpoint,
				Filesystem:  p.Fstype,
				Total:       usage.Total,
				Used:        usage.Used,
				Available:   usage.Free,
				UsedPercent: usage.UsedPercent,
			})
		}
	}

	// Network interfaces
	netIO, err := net.IOCounters(true)
	if err == nil {
		netInterfaces, _ := net.Interfaces()

		// Create a map of interface flags
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

		for _, io := range netIO {
			// Skip loopback and virtual interfaces
			if isVirtualInterface(io.Name) {
				continue
			}

			metrics.Network = append(metrics.Network, NetworkMetrics{
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
	}

	// Uptime & Load
	hostInfo, err := host.Info()
	if err == nil {
		metrics.Uptime = int64(hostInfo.Uptime)
	}

	loadAvg, err := load.Avg()
	if err == nil {
		metrics.LoadAvg = []float64{loadAvg.Load1, loadAvg.Load5, loadAvg.Load15}
	}

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

package metrics

import (
	"testing"
)

func TestGetSystemMetrics(t *testing.T) {
	metrics, err := GetSystemMetrics()
	if err != nil {
		t.Fatalf("GetSystemMetrics returned error: %v", err)
	}

	if metrics == nil {
		t.Fatal("GetSystemMetrics returned nil metrics")
	}

	// Test CPU metrics
	t.Run("CPU metrics", func(t *testing.T) {
		if metrics.CPU.UsagePercent < 0 || metrics.CPU.UsagePercent > 100 {
			t.Errorf("CPU usage percent should be between 0 and 100, got %f", metrics.CPU.UsagePercent)
		}

		if metrics.CPU.Cores <= 0 {
			t.Errorf("CPU cores should be > 0, got %d", metrics.CPU.Cores)
		}
	})

	// Test Memory metrics
	t.Run("Memory metrics", func(t *testing.T) {
		if metrics.Memory.Total == 0 {
			t.Error("Memory total should not be 0")
		}

		if metrics.Memory.UsedPercent < 0 || metrics.Memory.UsedPercent > 100 {
			t.Errorf("Memory used percent should be between 0 and 100, got %f", metrics.Memory.UsedPercent)
		}

		if metrics.Memory.Used > metrics.Memory.Total {
			t.Error("Memory used should not exceed total")
		}
	})

	// Test that Uptime is positive
	t.Run("Uptime", func(t *testing.T) {
		if metrics.Uptime <= 0 {
			t.Errorf("Uptime should be positive, got %d", metrics.Uptime)
		}
	})

	// Test LoadAvg (may be empty on some systems like Windows)
	t.Run("LoadAvg", func(t *testing.T) {
		// LoadAvg might not be available on all platforms, so just check structure
		if len(metrics.LoadAvg) > 0 {
			if len(metrics.LoadAvg) != 3 {
				t.Errorf("LoadAvg should have 3 values (1, 5, 15 min), got %d", len(metrics.LoadAvg))
			}
		}
	})
}

func TestGetSystemMetrics_Disks(t *testing.T) {
	metrics, err := GetSystemMetrics()
	if err != nil {
		t.Fatalf("GetSystemMetrics returned error: %v", err)
	}

	// Most systems should have at least one disk
	if len(metrics.Disks) == 0 {
		t.Log("Warning: No disks found (this might be expected in some containerized environments)")
		return
	}

	for i, disk := range metrics.Disks {
		t.Run(disk.MountPoint, func(t *testing.T) {
			if disk.MountPoint == "" {
				t.Errorf("Disk[%d] mount point should not be empty", i)
			}

			if disk.Total == 0 {
				t.Errorf("Disk[%d] total should not be 0", i)
			}

			if disk.UsedPercent < 0 || disk.UsedPercent > 100 {
				t.Errorf("Disk[%d] used percent should be between 0 and 100, got %f", i, disk.UsedPercent)
			}
		})
	}
}

func TestGetSystemMetrics_Network(t *testing.T) {
	metrics, err := GetSystemMetrics()
	if err != nil {
		t.Fatalf("GetSystemMetrics returned error: %v", err)
	}

	// Check network interfaces (may be filtered)
	for _, net := range metrics.Network {
		t.Run(net.Interface, func(t *testing.T) {
			if net.Interface == "" {
				t.Error("Network interface name should not be empty")
			}

			// Just verify these are non-negative (uint64 is always >= 0)
			// No specific value checks as they depend on system state
		})
	}
}

func TestIsVirtualFilesystem(t *testing.T) {
	tests := []struct {
		fstype   string
		expected bool
	}{
		{"ext4", false},
		{"xfs", false},
		{"ntfs", false},
		{"apfs", false},
		{"tmpfs", true},
		{"proc", true},
		{"sysfs", true},
		{"devtmpfs", true},
		{"cgroup", true},
		{"cgroup2", true},
		{"overlay", true},
		{"squashfs", true},
	}

	for _, tc := range tests {
		t.Run(tc.fstype, func(t *testing.T) {
			result := isVirtualFilesystem(tc.fstype)
			if result != tc.expected {
				t.Errorf("isVirtualFilesystem(%q) = %v, expected %v", tc.fstype, result, tc.expected)
			}
		})
	}
}

func TestIsVirtualInterface(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"lo", true},
		{"lo0", true},
		{"eth0", false},
		{"en0", false},
		{"wlan0", false},
		{"docker0", true},
		{"br-abc123", true},
		{"veth123abc", true},
		{"virbr0", true},
		{"flannel.1", true},
		{"cni0", true},
		{"calico123", true},
		{"weave", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isVirtualInterface(tc.name)
			if result != tc.expected {
				t.Errorf("isVirtualInterface(%q) = %v, expected %v", tc.name, result, tc.expected)
			}
		})
	}
}

func TestSystemMetricsStructure(t *testing.T) {
	// Test that the struct has all expected fields
	metrics := &SystemMetrics{
		CPU: CPUMetrics{
			UsagePercent: 50.0,
			Cores:        4,
			PerCore:      []float64{40.0, 50.0, 60.0, 50.0},
		},
		Memory: MemoryMetrics{
			Total:       16000000000,
			Used:        8000000000,
			Available:   8000000000,
			UsedPercent: 50.0,
			SwapTotal:   4000000000,
			SwapUsed:    1000000000,
		},
		Disks: []DiskMetrics{
			{
				Device:      "/dev/sda1",
				MountPoint:  "/",
				Filesystem:  "ext4",
				Total:       100000000000,
				Used:        50000000000,
				Available:   50000000000,
				UsedPercent: 50.0,
			},
		},
		Network: []NetworkMetrics{
			{
				Interface:   "eth0",
				BytesSent:   1000000,
				BytesRecv:   2000000,
				PacketsSent: 1000,
				PacketsRecv: 2000,
				ErrIn:       0,
				ErrOut:      0,
				DropIn:      0,
				DropOut:     0,
				IsUp:        true,
			},
		},
		Uptime:  86400,
		LoadAvg: []float64{1.0, 0.5, 0.25},
	}

	if metrics.CPU.UsagePercent != 50.0 {
		t.Error("CPU usage percent not set correctly")
	}
	if metrics.Memory.Total != 16000000000 {
		t.Error("Memory total not set correctly")
	}
	if len(metrics.Disks) != 1 {
		t.Error("Disks not set correctly")
	}
	if len(metrics.Network) != 1 {
		t.Error("Network not set correctly")
	}
	if metrics.Uptime != 86400 {
		t.Error("Uptime not set correctly")
	}
	if len(metrics.LoadAvg) != 3 {
		t.Error("LoadAvg not set correctly")
	}
}

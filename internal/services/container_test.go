package services

import (
	"context"
	"testing"
	"time"
)

func TestNewContainerService(t *testing.T) {
	// Test creating container service without audit service
	service := NewContainerService(nil)
	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Test creating container service with audit service
	auditService := &AuditService{}
	service = NewContainerService(auditService)
	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.audit != auditService {
		t.Error("expected audit service to be set")
	}
}

func TestContainerService_IsDockerAvailable(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// This test will pass/fail based on whether Docker is available
	// on the test machine. We just verify it doesn't panic.
	available := service.IsDockerAvailable(ctx)
	t.Logf("Docker available: %v", available)
}

func TestContainerService_List(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// List running containers
	containers, err := service.List(ctx, false)
	if err != nil {
		t.Fatalf("failed to list containers: %v", err)
	}
	t.Logf("Found %d running containers", len(containers))

	// List all containers (including stopped)
	allContainers, err := service.List(ctx, true)
	if err != nil {
		t.Fatalf("failed to list all containers: %v", err)
	}
	t.Logf("Found %d total containers", len(allContainers))

	// All containers count should be >= running containers count
	if len(allContainers) < len(containers) {
		t.Errorf("all containers count (%d) should be >= running containers count (%d)",
			len(allContainers), len(containers))
	}
}

func TestContainerInfo_Fields(t *testing.T) {
	// Test that ContainerInfo struct can be properly instantiated
	info := ContainerInfo{
		ID:      "abc123",
		Name:    "test-container",
		Image:   "nginx:latest",
		State:   "running",
		Status:  "Up 5 minutes",
		Created: time.Now(),
		Ports:   []PortMapping{},
	}

	if info.ID != "abc123" {
		t.Error("expected ID to be set")
	}
	if info.Name != "test-container" {
		t.Error("expected Name to be set")
	}
}

func TestPortMapping_Fields(t *testing.T) {
	pm := PortMapping{
		HostIP:        "0.0.0.0",
		HostPort:      "8080",
		ContainerPort: "80",
		Protocol:      "tcp",
	}

	if pm.HostIP != "0.0.0.0" {
		t.Error("expected HostIP to be set")
	}
	if pm.HostPort != "8080" {
		t.Error("expected HostPort to be set")
	}
	if pm.ContainerPort != "80" {
		t.Error("expected ContainerPort to be set")
	}
	if pm.Protocol != "tcp" {
		t.Error("expected Protocol to be set")
	}
}

func TestContainerConfig_Fields(t *testing.T) {
	config := ContainerConfig{
		Hostname:   "test-host",
		User:       "root",
		Env:        []string{"FOO=bar"},
		Cmd:        []string{"/bin/sh"},
		Entrypoint: []string{"/entrypoint.sh"},
		WorkingDir: "/app",
		Labels:     map[string]string{"app": "test"},
	}

	if config.Hostname != "test-host" {
		t.Error("expected Hostname to be set")
	}
	if config.User != "root" {
		t.Error("expected User to be set")
	}
	if len(config.Env) != 1 || config.Env[0] != "FOO=bar" {
		t.Error("expected Env to be set")
	}
}

func TestNetworkInfo_Fields(t *testing.T) {
	net := NetworkInfo{
		IPAddress:   "172.17.0.2",
		Gateway:     "172.17.0.1",
		MacAddress:  "02:42:ac:11:00:02",
		Networks:    map[string]string{"bridge": "172.17.0.2"},
		DNSServers:  []string{"8.8.8.8"},
		NetworkMode: "bridge",
	}

	if net.IPAddress != "172.17.0.2" {
		t.Error("expected IPAddress to be set")
	}
	if net.Gateway != "172.17.0.1" {
		t.Error("expected Gateway to be set")
	}
}

func TestMountInfo_Fields(t *testing.T) {
	mount := MountInfo{
		Type:        "bind",
		Source:      "/host/path",
		Destination: "/container/path",
		Mode:        "rw",
		RW:          true,
	}

	if mount.Type != "bind" {
		t.Error("expected Type to be set")
	}
	if mount.Source != "/host/path" {
		t.Error("expected Source to be set")
	}
	if !mount.RW {
		t.Error("expected RW to be true")
	}
}

func TestHealthStatus_Fields(t *testing.T) {
	health := HealthStatus{
		Status:        "healthy",
		FailingStreak: 0,
		Log:           "Health check passed",
	}

	if health.Status != "healthy" {
		t.Error("expected Status to be set")
	}
	if health.FailingStreak != 0 {
		t.Error("expected FailingStreak to be 0")
	}
}

func TestLogOptions_Fields(t *testing.T) {
	opts := LogOptions{
		Follow:     true,
		Tail:       "100",
		Timestamps: true,
		Since:      "2024-01-01",
		Until:      "2024-12-31",
	}

	if !opts.Follow {
		t.Error("expected Follow to be true")
	}
	if opts.Tail != "100" {
		t.Error("expected Tail to be set")
	}
	if !opts.Timestamps {
		t.Error("expected Timestamps to be true")
	}
}

func TestExecConfig_Fields(t *testing.T) {
	config := ExecConfig{
		Cmd:          []string{"ls", "-la"},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Env:          []string{"TERM=xterm"},
		WorkingDir:   "/app",
		User:         "root",
	}

	if len(config.Cmd) != 2 {
		t.Error("expected Cmd to have 2 elements")
	}
	if !config.AttachStdin {
		t.Error("expected AttachStdin to be true")
	}
	if !config.Tty {
		t.Error("expected Tty to be true")
	}
}

func TestExecResult_Fields(t *testing.T) {
	result := ExecResult{
		ExitCode: 0,
		Output:   "Hello, World!",
	}

	if result.ExitCode != 0 {
		t.Error("expected ExitCode to be 0")
	}
	if result.Output != "Hello, World!" {
		t.Error("expected Output to be set")
	}
}

func TestContainerDetail_Fields(t *testing.T) {
	detail := ContainerDetail{
		ContainerInfo: ContainerInfo{
			ID:    "abc123",
			Name:  "test",
			Image: "nginx",
			State: "running",
		},
		Config: ContainerConfig{
			Hostname: "test",
		},
		NetworkSettings: NetworkInfo{
			IPAddress: "172.17.0.2",
		},
		Mounts: []MountInfo{},
		HealthCheck: &HealthStatus{
			Status: "healthy",
		},
	}

	if detail.ID != "abc123" {
		t.Error("expected ID to be set from ContainerInfo")
	}
	if detail.Config.Hostname != "test" {
		t.Error("expected Config.Hostname to be set")
	}
	if detail.HealthCheck == nil {
		t.Error("expected HealthCheck to be set")
	}
}

// Integration tests that require Docker to be running
func TestContainerService_Get_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to get a non-existent container
	_, err := service.Get(ctx, "nonexistent-container-id-12345")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

func TestContainerService_Start_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to start a non-existent container
	err := service.Start(ctx, "nonexistent-container-id-12345", 1, "testuser")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

func TestContainerService_Stop_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to stop a non-existent container
	timeout := 10
	err := service.Stop(ctx, "nonexistent-container-id-12345", &timeout, 1, "testuser")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

func TestContainerService_Restart_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to restart a non-existent container
	timeout := 10
	err := service.Restart(ctx, "nonexistent-container-id-12345", &timeout, 1, "testuser")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

func TestContainerService_Remove_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to remove a non-existent container
	err := service.Remove(ctx, "nonexistent-container-id-12345", false, 1, "testuser")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

func TestContainerService_Logs_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to get logs from a non-existent container
	_, err := service.Logs(ctx, "nonexistent-container-id-12345", LogOptions{
		Tail: "100",
	})
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

func TestContainerService_CreateExec_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to create exec on a non-existent container
	_, err := service.CreateExec(ctx, "nonexistent-container-id-12345", ExecConfig{
		Cmd: []string{"/bin/sh"},
	})
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

func TestContainerService_Exec_NotFound(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to exec on a non-existent container
	_, err := service.Exec(ctx, "nonexistent-container-id-12345", ExecConfig{
		Cmd:          []string{"echo", "test"},
		AttachStdout: true,
		AttachStderr: true,
	}, 1, "testuser")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

// Test with nil timeout for Stop
func TestContainerService_Stop_NilTimeout(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to stop with nil timeout (should still fail because container doesn't exist)
	err := service.Stop(ctx, "nonexistent-container-id-12345", nil, 1, "testuser")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

// Test with nil timeout for Restart
func TestContainerService_Restart_NilTimeout(t *testing.T) {
	service := NewContainerService(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip test if Docker is not available
	if !service.IsDockerAvailable(ctx) {
		t.Skip("Docker is not available, skipping test")
	}

	// Try to restart with nil timeout (should still fail because container doesn't exist)
	err := service.Restart(ctx, "nonexistent-container-id-12345", nil, 1, "testuser")
	if err == nil {
		t.Error("expected error for non-existent container")
	}
}

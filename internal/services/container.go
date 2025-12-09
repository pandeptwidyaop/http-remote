// Package services provides business logic for the application.
package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// ContainerInfo represents a summary of a container for list view.
type ContainerInfo struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Image   string        `json:"image"`
	State   string        `json:"state"`
	Status  string        `json:"status"`
	Created time.Time     `json:"created"`
	Ports   []PortMapping `json:"ports"`
}

// ContainerDetail represents detailed information about a container.
type ContainerDetail struct {
	ContainerInfo
	Config          ContainerConfig `json:"config"`
	NetworkSettings NetworkInfo     `json:"network"`
	Mounts          []MountInfo     `json:"mounts"`
	HealthCheck     *HealthStatus   `json:"health_check,omitempty"`
}

// PortMapping represents a port mapping from host to container.
type PortMapping struct {
	HostIP        string `json:"host_ip"`
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// ContainerConfig represents container configuration.
type ContainerConfig struct {
	Hostname   string            `json:"hostname"`
	User       string            `json:"user"`
	Env        []string          `json:"env"`
	Cmd        []string          `json:"cmd"`
	Entrypoint []string          `json:"entrypoint"`
	WorkingDir string            `json:"working_dir"`
	Labels     map[string]string `json:"labels"`
}

// NetworkInfo represents container network settings.
type NetworkInfo struct {
	IPAddress   string            `json:"ip_address"`
	Gateway     string            `json:"gateway"`
	MacAddress  string            `json:"mac_address"`
	Networks    map[string]string `json:"networks"` // network name -> IP
	DNSServers  []string          `json:"dns_servers"`
	NetworkMode string            `json:"network_mode"`
}

// MountInfo represents a container mount/volume.
type MountInfo struct {
	Type        string `json:"type"` // bind, volume, tmpfs
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"` // rw, ro
	RW          bool   `json:"rw"`
}

// HealthStatus represents container health check status.
type HealthStatus struct {
	Status        string `json:"status"` // healthy, unhealthy, starting, none
	FailingStreak int    `json:"failing_streak"`
	Log           string `json:"log,omitempty"`
}

// LogOptions represents options for container log streaming.
type LogOptions struct {
	Follow     bool   `json:"follow"`
	Tail       string `json:"tail"`
	Timestamps bool   `json:"timestamps"`
	Since      string `json:"since"`
	Until      string `json:"until"`
}

// ContainerService handles container operations.
type ContainerService struct {
	audit *AuditService
}

// NewContainerService creates a new ContainerService instance.
func NewContainerService(audit *AuditService) *ContainerService {
	return &ContainerService{
		audit: audit,
	}
}

// getClient creates a new Docker client.
func (s *ContainerService) getClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

// IsDockerAvailable checks if Docker is available.
func (s *ContainerService) IsDockerAvailable(ctx context.Context) bool {
	cli, err := s.getClient()
	if err != nil {
		return false
	}
	defer func() { _ = cli.Close() }()

	_, err = cli.Ping(ctx)
	return err == nil
}

// List returns all containers.
func (s *ContainerService) List(ctx context.Context, all bool) ([]ContainerInfo, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	containers, err := cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		ports := make([]PortMapping, 0)
		for _, p := range c.Ports {
			ports = append(ports, PortMapping{
				HostIP:        p.IP,
				HostPort:      fmt.Sprintf("%d", p.PublicPort),
				ContainerPort: fmt.Sprintf("%d", p.PrivatePort),
				Protocol:      p.Type,
			})
		}

		result = append(result, ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Created: time.Unix(c.Created, 0),
			Ports:   ports,
		})
	}

	return result, nil
}

// Get returns detailed information about a container.
func (s *ContainerService) Get(ctx context.Context, containerID string) (*ContainerDetail, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Parse created time
	createdTime, _ := time.Parse(time.RFC3339Nano, inspect.Created)

	// Build port mappings from NetworkSettings
	ports := make([]PortMapping, 0)
	for portProto, bindings := range inspect.NetworkSettings.Ports {
		for _, binding := range bindings {
			ports = append(ports, PortMapping{
				HostIP:        binding.HostIP,
				HostPort:      binding.HostPort,
				ContainerPort: portProto.Port(),
				Protocol:      portProto.Proto(),
			})
		}
	}

	// Build network info
	networks := make(map[string]string)
	for name, net := range inspect.NetworkSettings.Networks {
		networks[name] = net.IPAddress
	}

	networkInfo := NetworkInfo{
		IPAddress:   inspect.NetworkSettings.IPAddress,
		Gateway:     inspect.NetworkSettings.Gateway,
		MacAddress:  inspect.NetworkSettings.MacAddress,
		Networks:    networks,
		DNSServers:  inspect.HostConfig.DNS,
		NetworkMode: string(inspect.HostConfig.NetworkMode),
	}

	// Build mount info
	mounts := make([]MountInfo, 0, len(inspect.Mounts))
	for _, m := range inspect.Mounts {
		mode := "rw"
		if !m.RW {
			mode = "ro"
		}
		mounts = append(mounts, MountInfo{
			Type:        string(m.Type),
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        mode,
			RW:          m.RW,
		})
	}

	// Health check status
	var healthStatus *HealthStatus
	if inspect.State.Health != nil {
		healthStatus = &HealthStatus{
			Status:        inspect.State.Health.Status,
			FailingStreak: inspect.State.Health.FailingStreak,
		}
		if len(inspect.State.Health.Log) > 0 {
			lastLog := inspect.State.Health.Log[len(inspect.State.Health.Log)-1]
			healthStatus.Log = lastLog.Output
		}
	}

	detail := &ContainerDetail{
		ContainerInfo: ContainerInfo{
			ID:      inspect.ID[:12],
			Name:    strings.TrimPrefix(inspect.Name, "/"),
			Image:   inspect.Config.Image,
			State:   inspect.State.Status,
			Status:  fmt.Sprintf("%s (%s)", inspect.State.Status, inspect.State.StartedAt),
			Created: createdTime,
			Ports:   ports,
		},
		Config: ContainerConfig{
			Hostname:   inspect.Config.Hostname,
			User:       inspect.Config.User,
			Env:        inspect.Config.Env,
			Cmd:        inspect.Config.Cmd,
			Entrypoint: inspect.Config.Entrypoint,
			WorkingDir: inspect.Config.WorkingDir,
			Labels:     inspect.Config.Labels,
		},
		NetworkSettings: networkInfo,
		Mounts:          mounts,
		HealthCheck:     healthStatus,
	}

	return detail, nil
}

// Start starts a container.
func (s *ContainerService) Start(ctx context.Context, containerID string, userID int64, username string) error {
	cli, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Audit log
	if s.audit != nil {
		_ = s.audit.Log(AuditLog{
			UserID:       &userID,
			Username:     username,
			Action:       "container_start",
			ResourceType: "container",
			ResourceID:   containerID,
		})
	}

	return nil
}

// Stop stops a container.
func (s *ContainerService) Stop(ctx context.Context, containerID string, timeout *int, userID int64, username string) error {
	cli, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	stopOptions := container.StopOptions{}
	if timeout != nil {
		stopOptions.Timeout = timeout
	}

	if err := cli.ContainerStop(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Audit log
	if s.audit != nil {
		_ = s.audit.Log(AuditLog{
			UserID:       &userID,
			Username:     username,
			Action:       "container_stop",
			ResourceType: "container",
			ResourceID:   containerID,
		})
	}

	return nil
}

// Restart restarts a container.
func (s *ContainerService) Restart(ctx context.Context, containerID string, timeout *int, userID int64, username string) error {
	cli, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	stopOptions := container.StopOptions{}
	if timeout != nil {
		stopOptions.Timeout = timeout
	}

	if err := cli.ContainerRestart(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// Audit log
	if s.audit != nil {
		_ = s.audit.Log(AuditLog{
			UserID:       &userID,
			Username:     username,
			Action:       "container_restart",
			ResourceType: "container",
			ResourceID:   containerID,
		})
	}

	return nil
}

// Remove removes a container.
func (s *ContainerService) Remove(ctx context.Context, containerID string, force bool, userID int64, username string) error {
	cli, err := s.getClient()
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Audit log
	if s.audit != nil {
		var details map[string]interface{}
		if force {
			details = map[string]interface{}{"force": true}
		}
		_ = s.audit.Log(AuditLog{
			UserID:       &userID,
			Username:     username,
			Action:       "container_remove",
			ResourceType: "container",
			ResourceID:   containerID,
			Details:      details,
		})
	}

	return nil
}

// Logs returns container logs as a reader.
func (s *ContainerService) Logs(ctx context.Context, containerID string, opts LogOptions) (io.ReadCloser, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	// Note: client will be closed when the returned reader is closed

	logOptions := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
		Since:      opts.Since,
		Until:      opts.Until,
	}

	reader, err := cli.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	// Wrap reader to close client when done
	return &logReaderWrapper{
		reader: reader,
		client: cli,
	}, nil
}

// logReaderWrapper wraps a log reader to close the client when done.
type logReaderWrapper struct {
	reader io.ReadCloser
	client *client.Client
}

func (w *logReaderWrapper) Read(p []byte) (n int, err error) {
	return w.reader.Read(p)
}

func (w *logReaderWrapper) Close() error {
	_ = w.reader.Close()
	return w.client.Close()
}

// StreamLogs streams container logs line by line.
func (s *ContainerService) StreamLogs(ctx context.Context, containerID string, opts LogOptions, lineFn func(line string)) error {
	reader, err := s.Logs(ctx, containerID, opts)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Docker log stream has 8-byte header for each log entry
	// We need to strip it for clean output
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scanner.Text()
			// Docker multiplexed stream has 8-byte header
			// Header: [STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4]
			if len(line) > 8 {
				line = line[8:]
			}
			lineFn(line)
		}
	}

	return scanner.Err()
}

// ExecConfig represents configuration for container exec.
type ExecConfig struct {
	Cmd          []string `json:"cmd"`
	AttachStdin  bool     `json:"attach_stdin"`
	AttachStdout bool     `json:"attach_stdout"`
	AttachStderr bool     `json:"attach_stderr"`
	Tty          bool     `json:"tty"`
	Env          []string `json:"env,omitempty"`
	WorkingDir   string   `json:"working_dir,omitempty"`
	User         string   `json:"user,omitempty"`
}

// ExecResult represents the result of an exec command.
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

// Exec executes a command in a container and returns the result.
func (s *ContainerService) Exec(ctx context.Context, containerID string, config ExecConfig, userID int64, username string) (*ExecResult, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	execConfig := container.ExecOptions{
		Cmd:          config.Cmd,
		AttachStdin:  config.AttachStdin,
		AttachStdout: config.AttachStdout,
		AttachStderr: config.AttachStderr,
		Tty:          config.Tty,
		Env:          config.Env,
		WorkingDir:   config.WorkingDir,
		User:         config.User,
	}

	execID, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := cli.ContainerExecAttach(ctx, execID.ID, container.ExecAttachOptions{Tty: config.Tty})
	if err != nil {
		return nil, fmt.Errorf("failed to attach exec: %w", err)
	}
	defer resp.Close()

	// Read output
	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read exec output: %w", err)
	}

	// Get exit code
	inspectResp, err := cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect exec: %w", err)
	}

	// Audit log
	if s.audit != nil {
		cmdStr := strings.Join(config.Cmd, " ")
		_ = s.audit.Log(AuditLog{
			UserID:       &userID,
			Username:     username,
			Action:       "container_exec",
			ResourceType: "container",
			ResourceID:   containerID,
			Details:      map[string]interface{}{"cmd": cmdStr},
		})
	}

	return &ExecResult{
		ExitCode: inspectResp.ExitCode,
		Output:   string(output),
	}, nil
}

// CreateExec creates an exec instance and returns the exec ID.
func (s *ContainerService) CreateExec(ctx context.Context, containerID string, config ExecConfig) (string, error) {
	cli, err := s.getClient()
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer func() { _ = cli.Close() }()

	execConfig := container.ExecOptions{
		Cmd:          config.Cmd,
		AttachStdin:  config.AttachStdin,
		AttachStdout: config.AttachStdout,
		AttachStderr: config.AttachStderr,
		Tty:          config.Tty,
		Env:          config.Env,
		WorkingDir:   config.WorkingDir,
		User:         config.User,
	}

	execID, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	return execID.ID, nil
}

// AttachExec attaches to an exec instance and returns the hijacked connection.
func (s *ContainerService) AttachExec(ctx context.Context, execID string) (types.HijackedResponse, error) {
	cli, err := s.getClient()
	if err != nil {
		return types.HijackedResponse{}, fmt.Errorf("failed to create Docker client: %w", err)
	}
	// Note: client will be closed by the caller

	return cli.ContainerExecAttach(ctx, execID, container.ExecAttachOptions{Tty: true})
}

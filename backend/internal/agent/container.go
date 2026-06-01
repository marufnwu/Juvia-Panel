package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ContainerManager handles Docker container operations
type ContainerManager struct {
	dataDir     string
	networkName string
}

// ContainerInfo holds detailed container information
type ContainerInfo struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Status     string            `json:"status"`
	Created    time.Time         `json:"created"`
	Ports      []PortMapping     `json:"ports"`
	EnvVars    map[string]string `json:"env_vars"`
	Volumes    []VolumeMount     `json:"volumes"`
	Network    string            `json:"network"`
	State      string             `json:"state"` // running, exited, paused
	ExitCode   int               `json:"exit_code,omitempty"`
}

// NewContainerManager creates a new ContainerManager
func NewContainerManager() *ContainerManager {
	return &ContainerManager{
		dataDir:     "/var/panel",
		networkName: "panel_apps",
	}
}

// SetDataDir sets the data directory
func (cm *ContainerManager) SetDataDir(dir string) {
	cm.dataDir = dir
}

// SetNetworkName sets the Docker network name
func (cm *ContainerManager) SetNetworkName(name string) {
	cm.networkName = name
}

// CreateAndStart creates and starts a container
func (cm *ContainerManager) CreateAndStart(ctx context.Context, params RunParams) (*RunResult, error) {
	// Ensure network exists
	if err := cm.EnsureNetwork(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure network: %w", err)
	}

	// Determine port allocation
	externalPort := params.Ports[0].External
	if externalPort == 0 {
		// Auto-assign port from range 3000-3999
		var err error
		externalPort, err = cm.allocatePort()
		if err != nil {
			return nil, fmt.Errorf("failed to allocate port: %w", err)
		}
	}

	// Build docker run command
	args := []string{"run", "-d"}

	// Add container name
	containerName := fmt.Sprintf("panel-%s", params.AppID)
	args = append(args, "--name", containerName)

	// Add network
	args = append(args, "--network", cm.networkName)

	// Add restart policy
	restartPolicy := params.Restart
	if restartPolicy == "" {
		restartPolicy = "unless-stopped"
	}
	args = append(args, "--restart", restartPolicy)

	// Add environment variables
	for key, value := range params.EnvVars {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add port mapping
	internalPort := params.Ports[0].Internal
	if internalPort == 0 {
		internalPort = 3000 // default
	}
	args = append(args, "-p", fmt.Sprintf("%d:%d", externalPort, internalPort))

	// Add memory limit
	if params.MemoryLimit != "" {
		args = append(args, "-m", params.MemoryLimit)
	}

	// Add CPU quota
	if params.CPUQuota > 0 {
		args = append(args, "--cpu-quota", strconv.FormatInt(params.CPUQuota, 10))
	}

	// Add volumes
	for _, vol := range params.Volumes {
		readOnly := ""
		if vol.ReadOnly {
			readOnly = ":ro"
		}
		args = append(args, "-v", fmt.Sprintf("%s:%s%s", vol.HostPath, vol.ContainerPath, readOnly))
	}

	// Add image
	args = append(args, params.Image)

	// Execute docker run
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker run failed: %w - %s", err, string(output))
	}

	// Extract container ID
	containerID := strings.TrimSpace(string(output))
	if len(containerID) > 12 {
		containerID = containerID[:12]
	}

	return &RunResult{
		ContainerID: containerID,
		Image:       params.Image,
		Status:      "running",
		Port:        externalPort,
	}, nil
}

// StopContainer stops a container gracefully
func (cm *ContainerManager) StopContainer(ctx context.Context, containerID string, timeout int) error {
	if timeout <= 0 {
		timeout = 10 // default 10 seconds
	}

	args := []string{"stop", "-t", strconv.Itoa(timeout), containerID}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if container is already stopped
		if strings.Contains(string(output), "No such container") {
			return nil // already stopped, consider it success
		}
		return fmt.Errorf("docker stop failed: %w - %s", err, string(output))
	}

	return nil
}

// RestartContainer restarts a container
func (cm *ContainerManager) RestartContainer(ctx context.Context, containerID string) error {
	args := []string{"restart", containerID}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker restart failed: %w - %s", err, string(output))
	}
	return nil
}

// StartContainer starts a stopped container
func (cm *ContainerManager) StartContainer(ctx context.Context, containerID string) error {
	args := []string{"start", containerID}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker start failed: %w - %s", err, string(output))
	}
	return nil
}

// GetContainerLogs retrieves container logs
func (cm *ContainerManager) GetContainerLogs(ctx context.Context, containerID string, stream string, tail int) ([]LogLine, error) {
	args := []string{"logs"}

	if tail > 0 {
		args = append(args, "--tail", strconv.Itoa(tail))
	} else {
		args = append(args, "--tail", "100")
	}

	// Add timestamps
	args = append(args, "-t")

	// Specify stream
	if stream == "stdout" {
		args = append(args, "--stdout")
	} else if stream == "stderr" {
		args = append(args, "--stderr")
	} else {
		// both - docker defaults to both
	}

	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Logs command may fail if container doesn't exist
		if strings.Contains(string(output), "No such container") {
			return []LogLine{}, nil
		}
		return nil, fmt.Errorf("docker logs failed: %w - %s", err, string(output))
	}

	// Parse logs
	return parseDockerLogs(string(output)), nil
}

// ExecInContainer executes a command in a running container
func (cm *ContainerManager) ExecInContainer(ctx context.Context, containerID string, command []string, workDir string) (*ExecResult, error) {
	args := []string{"exec"}

	if workDir != "" {
		args = append(args, "-w", workDir)
	}

	args = append(args, containerID)
	args = append(args, command...)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	exitCode := 0

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return &ExecResult{
		ExitCode: exitCode,
		Output:   string(output),
	}, nil
}

// GetContainerStats retrieves container resource usage
func (cm *ContainerManager) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	args := []string{"stats", "--no-stream", "--format", "{{.CPUPerc}},{{.MemUsage}}", containerID}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "No such container") {
			return &ContainerStats{}, nil
		}
		return nil, fmt.Errorf("docker stats failed: %w - %s", err, string(output))
	}

	stats := &ContainerStats{}

	// Parse output: "12.5%,256MiB / 512MiB"
	parts := strings.Split(string(output), ",")
	if len(parts) >= 2 {
		// CPU percentage
		cpuStr := strings.TrimSuffix(parts[0], "%")
		if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
			stats.CPUPercent = cpu
		}

		// Memory usage
		memPart := parts[1]
		memParts := strings.Split(memPart, "/")
		if len(memParts) >= 2 {
			// Parse used memory (e.g., "256MiB")
			memUsed := strings.TrimSpace(memParts[0])
			if mb, err := parseMemoryToMB(memUsed); err == nil {
				stats.MemoryUsageMB = mb
			}

			// Parse memory limit
			memLimit := strings.TrimSpace(memParts[1])
			if mb, err := parseMemoryToMB(memLimit); err == nil {
				stats.MemoryLimitMB = mb
			}
		}
	}

	// Get network I/O
	netStats, err := cm.getNetworkStats(ctx, containerID)
	if err == nil {
		stats.NetworkRX = netStats.RX
		stats.NetworkTX = netStats.TX
	}

	return stats, nil
}

// RemoveContainer removes a container
func (cm *ContainerManager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	args := []string{"rm"}

	if force {
		args = append(args, "-f")
	}

	args = append(args, containerID)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if container doesn't exist
		if strings.Contains(string(output), "No such container") ||
			strings.Contains(string(output), "already removed") {
			return nil // already removed, consider it success
		}
		return fmt.Errorf("docker rm failed: %w - %s", err, string(output))
	}

	return nil
}

// EnsureNetwork creates the panel_apps network if it doesn't exist
func (cm *ContainerManager) EnsureNetwork(ctx context.Context) error {
	// Check if network exists
	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--format", "{{.Name}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		if scanner.Text() == cm.networkName {
			return nil // network exists
		}
	}

	// Create network
	createCmd := exec.CommandContext(ctx, "docker", "network", "create", cm.networkName)
	_, err = createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	return nil
}

// allocatePort finds an available port in the range 3000-3999
func (cm *ContainerManager) allocatePort() (int, error) {
	// Use docker to find ports in use
	cmd := exec.Command("docker", "ps", "--format", "{{.Ports}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to default port
		return 3000, nil
	}

	usedPorts := make(map[int]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		ports := scanner.Text()
		// Parse port mappings like "0.0.0.0:3000->3000/tcp"
		parts := strings.Split(ports, "->")
		if len(parts) >= 2 {
			hostPart := parts[0]
			hostParts := strings.Split(hostPart, ":")
			if len(hostParts) >= 2 {
				if port, err := strconv.Atoi(hostParts[len(hostParts)-1]); err == nil {
					if port >= 3000 && port <= 3999 {
						usedPorts[port] = true
					}
				}
			}
		}
	}

	// Find first available port
	for port := 3000; port <= 3999; port++ {
		if !usedPorts[port] {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range 3000-3999")
}

// parseMemoryToMB parses a memory string like "256MiB" to megabytes
func parseMemoryToMB(memStr string) (int, error) {
	memStr = strings.TrimSpace(memStr)
	memStr = strings.ToUpper(memStr)

	// Remove common suffixes and parse number
	multiplier := 1
	if strings.HasSuffix(memStr, "GIB") || strings.HasSuffix(memStr, "GB") {
		multiplier = 1024
		memStr = strings.TrimSuffix(memStr, "GIB")
		memStr = strings.TrimSuffix(memStr, "GB")
	} else if strings.HasSuffix(memStr, "MIB") || strings.HasSuffix(memStr, "MB") {
		multiplier = 1
		memStr = strings.TrimSuffix(memStr, "MIB")
		memStr = strings.TrimSuffix(memStr, "MB")
	} else if strings.HasSuffix(memStr, "KIB") || strings.HasSuffix(memStr, "KB") {
		multiplier = 1 / 1024
		memStr = strings.TrimSuffix(memStr, "KIB")
		memStr = strings.TrimSuffix(memStr, "KB")
	}

	memStr = strings.TrimSpace(memStr)
	value, err := strconv.ParseFloat(memStr, 64)
	if err != nil {
		return 0, err
	}

	return int(value * float64(multiplier)), nil
}

type networkStats struct {
	RX int64
	TX int64
}

// getNetworkStats retrieves network I/O for a container
func (cm *ContainerManager) getNetworkStats(ctx context.Context, containerID string) (*networkStats, error) {
	args := []string{"inspect", "--format", "{{json .NetworkSettings.Networks}}", containerID}
	cmd := exec.CommandContext(ctx, "docker", args...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// Parse the JSON output to get network stats
	// This is a simplified version; real implementation would parse more thoroughly
	var stats networkStats

	// For now, return zero stats
	return &stats, nil
}

// parseDockerLogs parses docker logs output into LogLine structs
func parseDockerLogs(output string) []LogLine {
	var logs []LogLine
	scanner := bufio.NewScanner(strings.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()

		// Docker log format: "2024-06-01T12:34:56.123456789Z some message"
		// Or with stream: "2024-06-01T12:34:56.123456789Z stdout some message"

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		timestamp := parts[0]
		message := parts[1]

		// Check for stream prefix
		stream := "stdout"
		if strings.HasPrefix(message, "stdout ") {
			message = strings.TrimPrefix(message, "stdout ")
		} else if strings.HasPrefix(message, "stderr ") {
			stream = "stderr"
			message = strings.TrimPrefix(message, "stderr ")
		}

		logs = append(logs, LogLine{
			Timestamp: timestamp,
			Level:     "info",
			Message:   message,
			Stream:    stream,
		})
	}

	return logs
}

// GetContainerStatus returns the current status of a container
func (cm *ContainerManager) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	args := []string{"inspect", "--format", "{{.State.Status}}", containerID}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "No such container") {
			return "removed", nil
		}
		return "", fmt.Errorf("docker inspect failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetContainerInfo returns detailed container information
func (cm *ContainerManager) GetContainerInfo(ctx context.Context, containerID string) (map[string]interface{}, error) {
	args := []string{"inspect", "--format", "{{json .}}", containerID}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w", err)
	}

	var info map[string]interface{}
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse docker output: %w", err)
	}

	return info, nil
}

// StreamLogs streams logs from a container in real-time
func (cm *ContainerManager) StreamLogs(ctx context.Context, containerID string, tail int) (<-chan LogLine, error) {
	args := []string{"logs", "-f"}

	if tail > 0 {
		args = append(args, "--tail", strconv.Itoa(tail))
	} else {
		args = append(args, "--tail", "100")
	}

	args = append(args, "-t", containerID)

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	logChan := make(chan LogLine)

	go func() {
		defer close(logChan)
		defer cmd.Wait()

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, " ", 2)
			if len(parts) < 2 {
				continue
			}

			timestamp := parts[0]
			message := parts[1]

			stream := "stdout"
			if strings.HasPrefix(message, "stderr ") {
				stream = "stderr"
				message = strings.TrimPrefix(message, "stderr ")
			}

			select {
			case logChan <- LogLine{
				Timestamp: timestamp,
				Level:     "info",
				Message:   message,
				Stream:    stream,
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return logChan, nil
}

// ListContainers returns all containers managed by panel
func (cm *ContainerManager) ListContainers(ctx context.Context) ([]*ContainerInfo, error) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Run docker ps with format to get all containers
	args := []string{"ps", "-a", "--format", "{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.CreatedAt}}|{{.Ports}}"}

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w - %s", err, string(output))
	}

	var containers []*ContainerInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		container := &ContainerInfo{
			ID:      parts[0],
			Name:    parts[1],
			Image:   parts[2],
			Status:  parts[3],
			Created: parseCreatedAt(parts[4]),
			Ports:   parsePorts(parts[5]),
			State:   "unknown",
		}

		// Get detailed info for running containers
		if strings.HasPrefix(container.Status, "Up") {
			container.State = "running"
		} else if strings.Contains(container.Status, "Exited") {
			container.State = "exited"
		}

		containers = append(containers, container)
	}

	return containers, nil
}

// parseCreatedAt parses Docker's created at timestamp
func parseCreatedAt(s string) time.Time {
	// Docker format: "2024-06-01 12:34:56 +0000 UTC"
	layouts := []string{
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 -0700 UTC",
		time.RFC3339,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Now()
}

// parsePorts parses Docker port output
func parsePorts(s string) []PortMapping {
	var ports []PortMapping
	if s == "" {
		return ports
	}

	// Port format: "0.0.0.0:3000->3000/tcp, 0.0.0.0:8080->80/tcp"
	portMappings := strings.Split(s, ", ")
	for _, pm := range portMappings {
		// Parse "0.0.0.0:3000->3000/tcp"
		parts := strings.Split(pm, "->")
		if len(parts) != 2 {
			continue
		}

		// Host part: "0.0.0.0:3000"
		hostParts := strings.Split(parts[0], ":")
		if len(hostParts) < 2 {
			continue
		}
		externalPort, err := strconv.Atoi(hostParts[len(hostParts)-1])
		if err != nil {
			continue
		}

		// Container part: "3000/tcp"
		containerPart := strings.Split(parts[1], "/")
		internalPort, err := strconv.Atoi(containerPart[0])
		if err != nil {
			continue
		}

		ports = append(ports, PortMapping{
			Internal: internalPort,
			External: externalPort,
		})
	}

	return ports
}

// Stats holds resource statistics for a container
type Stats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryUsageMB int     `json:"memory_usage_mb"`
	MemoryLimitMB int     `json:"memory_limit_mb"`
	NetworkRXMB   float64 `json:"network_rx_mb"`
	NetworkTXMB   float64 `json:"network_tx_mb"`
	BlockReadMB   float64 `json:"block_read_mb"`
	BlockWriteMB   float64 `json:"block_write_mb"`
}

// GetStats retrieves resource usage statistics for a container
func (cm *ContainerManager) GetStats(ctx context.Context, id string) (*Stats, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Get stats in JSON format
	args := []string{"stats", "--no-stream", "--format", "{{json .}}", id}
	cmd := exec.CommandContext(ctx, "docker", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "No such container") ||
			strings.Contains(string(output), "not found") {
			return nil, fmt.Errorf("container not found")
		}
		return nil, fmt.Errorf("failed to get stats: %w - %s", err, string(output))
	}

	// Parse JSON output from docker stats
	var rawStats struct {
		CPUPerc    string `json:"CPUPerc"`
		MemUsage  string `json:"MemUsage"`
		MemPerc   string `json:"MemPerc"`
		NetIO     string `json:"NetIO"`
		BlockIO   string `json:"BlockIO"`
		PIDs      string `json:"PIDs"`
	}

	if err := json.Unmarshal(output, &rawStats); err != nil {
		// Fallback to parsing raw format
		return cm.parseStatsOutput(string(output))
	}

	stats := &Stats{}

	// Parse CPU percentage
	cpuStr := strings.TrimSuffix(rawStats.CPUPerc, "%")
	if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
		stats.CPUPercent = cpu
	}

	// Parse memory usage "256MiB / 512MiB"
	memParts := strings.Split(rawStats.MemUsage, "/")
	if len(memParts) >= 2 {
		if mb, err := parseMemoryToMB(memParts[0]); err == nil {
			stats.MemoryUsageMB = mb
		}
		if mb, err := parseMemoryToMB(memParts[1]); err == nil {
			stats.MemoryLimitMB = mb
		}
	}

	// Parse memory percentage
	memPctStr := strings.TrimSuffix(rawStats.MemPerc, "%")
	if memPct, err := strconv.ParseFloat(memPctStr, 64); err == nil {
		stats.MemoryPercent = memPct
	}

	// Parse network I/O "123MB / 456MB"
	netParts := strings.Split(rawStats.NetIO, "/")
	if len(netParts) >= 2 {
		if mb, err := parseNetworkBytes(netParts[0]); err == nil {
			stats.NetworkRXMB = mb
		}
		if mb, err := parseNetworkBytes(netParts[1]); err == nil {
			stats.NetworkTXMB = mb
		}
	}

	// Parse block I/O "1MB / 2MB"
	blockParts := strings.Split(rawStats.BlockIO, "/")
	if len(blockParts) >= 2 {
		if mb, err := parseNetworkBytes(blockParts[0]); err == nil {
			stats.BlockReadMB = mb
		}
		if mb, err := parseNetworkBytes(blockParts[1]); err == nil {
			stats.BlockWriteMB = mb
		}
	}

	return stats, nil
}

// parseStatsOutput parses raw docker stats output as fallback
func (cm *ContainerManager) parseStatsOutput(output string) (*Stats, error) {
	stats := &Stats{}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return stats, fmt.Errorf("unexpected stats output format")
	}

	// Second line contains values
	values := lines[1]

	// Parse CPU: "12.34%"
	cpuIdx := strings.Index(values, "%")
	if cpuIdx > 0 {
		start := strings.LastIndex(values[:cpuIdx], " ")
		if start > 0 {
			if cpu, err := strconv.ParseFloat(values[start+1:cpuIdx], 64); err == nil {
				stats.CPUPercent = cpu
			}
		}
	}

	return stats, nil
}

// RunContainer is a convenience wrapper that runs a container with the given parameters
// and returns the container ID, status, and assigned port
func (cm *ContainerManager) RunContainer(ctx context.Context, image string, name string, envVars map[string]string, volumes []VolumeMount, ports []PortMapping, network string, memoryLimit string) (*RunResult, error) {
	params := RunParams{
		AppID:       name,
		Image:       image,
		EnvVars:     envVars,
		Volumes:     volumes,
		Ports:       ports,
		Network:     network,
		Restart:     "unless-stopped",
		MemoryLimit: memoryLimit,
	}
	return cm.CreateAndStart(ctx, params)
}

// parseNetworkBytes parses network byte strings like "123MB" or "1.5GB"
func parseNetworkBytes(s string) (float64, error) {
	s = strings.TrimSpace(s)

	multiplier := 1.0
	if strings.HasSuffix(s, "GB") {
		multiplier = 1024.0
		s = strings.TrimSuffix(s, "GB")
	} else if strings.HasSuffix(s, "MB") {
		multiplier = 1.0
		s = strings.TrimSuffix(s, "MB")
	} else if strings.HasSuffix(s, "KB") {
		multiplier = 1.0 / 1024.0
		s = strings.TrimSuffix(s, "KB")
	} else if strings.HasSuffix(s, "B") {
		multiplier = 1.0 / (1024.0 * 1024.0)
		s = strings.TrimSuffix(s, "B")
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}

	return value * multiplier, nil
}
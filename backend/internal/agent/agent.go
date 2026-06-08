// Package agent provides the agent daemon that handles Docker operations
// via Unix socket communication. The agent runs as a separate process
// and communicates with the API server through JSON messages.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Command types supported by the agent
const (
	CmdPing          = "ping"
	CmdBuild         = "build"
	CmdRun           = "run"
	CmdStop          = "stop"
	CmdStart         = "start"
	CmdRestart       = "restart"
	CmdLogs          = "logs"
	CmdExec          = "exec"
	CmdStats         = "stats"
	CmdRemove        = "remove"
	CmdHealthCheck   = "healthcheck"
)

// Default socket path
const DefaultSocketPath = "/var/run/panel/agent.sock"

// DefaultAgentTCPPort is the TCP port the agent listens on when Unix sockets are
// not available (e.g., Windows or container environments). It must not collide
// with the API port (default 9090).
const DefaultAgentTCPPort = 9091

// Agent handles Unix socket communication and command execution
type Agent struct {
	socketPath string
	listener   net.Listener
	buildMgr   *BuildManager
	containers *ContainerManager
	network    *NetworkManager
	health     *HealthChecker
	mu         sync.RWMutex
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// Command represents an incoming agent command
type Command struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id"`
	Params    json.RawMessage `json:"params"`
}

// Response represents an agent response
type Response struct {
	RequestID string          `json:"request_id"`
	Status    string          `json:"status"`
	Data      interface{}     `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// New creates a new Agent instance
func New(socketPath string) *Agent {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Agent{
		socketPath:  socketPath,
		buildMgr:    NewBuildManager(),
		containers:  NewContainerManager(),
		network:     NewNetworkManager(),
		health:      NewHealthChecker(),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins listening on the Unix socket or TCP port depending on OS
func (a *Agent) Start() error {
	dir := filepath.Dir(a.socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	os.Remove(a.socketPath)

	var listener net.Listener
	var err error

	if runtime.GOOS == "windows" {
		// On Windows use TCP; port must not collide with the API (default 9090).
		// Use PANEL_AGENT_TCP_PORT env var or DefaultAgentTCPPort (9091).
		port := os.Getenv("PANEL_AGENT_TCP_PORT")
		if port == "" {
			port = fmt.Sprintf("%d", DefaultAgentTCPPort)
		}
		listener, err = net.Listen("tcp", "127.0.0.1:"+port)
		if err != nil {
			return fmt.Errorf("failed to listen on TCP port %s: %w", port, err)
		}
		log.Printf("Agent listening on TCP 127.0.0.1:%s (Windows mode)", port)
	} else {
		listener, err = net.Listen("unix", a.socketPath)
		if err != nil {
			return fmt.Errorf("failed to listen on socket: %w", err)
		}
		os.Chmod(a.socketPath, 0660)
		log.Printf("Agent listening on %s", a.socketPath)
	}

	a.listener = listener
	go a.acceptLoop()

	return nil
}

// acceptLoop handles incoming socket connections
func (a *Agent) acceptLoop() {
	for {
		select {
		case <-a.ctx.Done():
			return
		default:
			conn, err := a.listener.Accept()
			if err != nil {
				if a.ctx.Err() != nil {
					return
				}
				log.Printf("Accept error: %v", err)
				continue
			}
			a.wg.Add(1)
			go a.handleConnection(conn)
		}
	}
}

// handleConnection processes a single client connection
func (a *Agent) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		a.wg.Done()
	}()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var cmd Command
		if err := decoder.Decode(&cmd); err != nil {
			if err.Error() != "EOF" {
				log.Printf("Decode error: %v", err)
			}
			return
		}

		resp := a.processCommand(cmd)
		if err := encoder.Encode(resp); err != nil {
			log.Printf("Encode error: %v", err)
			return
		}
	}
}

// processCommand handles an agent command and returns response
func (a *Agent) processCommand(cmd Command) Response {
	var resp Response
	resp.RequestID = cmd.RequestID

	switch cmd.Type {
	case CmdPing:
		resp.Status = "ok"
		resp.Data = map[string]string{"status": "ok"}

	case CmdBuild:
		var params BuildParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		result, err := a.buildMgr.Build(a.ctx, params)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = result
		}

	case CmdRun:
		var params RunParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		result, err := a.containers.CreateAndStart(a.ctx, params)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = result
		}

	case CmdStop:
		var params StopParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		err := a.containers.StopContainer(a.ctx, params.ContainerID, params.Timeout)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = map[string]string{"status": "stopped"}
		}

	case CmdStart:
		var params StartParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		err := a.containers.StartContainer(a.ctx, params.ContainerID)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = map[string]string{"status": "started"}
		}

	case CmdRestart:
		var params RestartParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		err := a.containers.RestartContainer(a.ctx, params.ContainerID)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = map[string]string{"status": "restarted"}
		}

	case CmdLogs:
		var params LogsParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		logs, err := a.containers.GetContainerLogs(a.ctx, params.ContainerID, params.Stream, params.Tail)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = logs
		}

	case CmdExec:
		var params ExecParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		result, err := a.containers.ExecInContainer(a.ctx, params.ContainerID, params.Command, params.WorkDir)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = result
		}

	case CmdStats:
		var params StatsParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		stats, err := a.containers.GetContainerStats(a.ctx, params.ContainerID)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = stats
		}

	case CmdRemove:
		var params RemoveParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		err := a.containers.RemoveContainer(a.ctx, params.ContainerID, params.Force)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = map[string]string{"status": "removed"}
		}

	case CmdHealthCheck:
		var params HealthCheckParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		a.health.Start(a.ctx, params)
		resp.Status = "success"
		resp.Data = map[string]string{"status": "health_check_started"}

	case "list":
		// ListContainers - list all containers
		containers, err := a.containers.ListContainers(a.ctx)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = containers
		}

	case "get-stats":
		// GetStats - get container resource stats
		var params StatsParams
		if err := json.Unmarshal(cmd.Params, &params); err != nil {
			resp.Status = "error"
			resp.Error = fmt.Sprintf("invalid params: %v", err)
			return resp
		}
		stats, err := a.containers.GetStats(a.ctx, params.ContainerID)
		if err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		} else {
			resp.Status = "success"
			resp.Data = stats
		}

	default:
		resp.Status = "error"
		resp.Error = fmt.Sprintf("unknown command: %s", cmd.Type)
	}

	return resp
}

// Stop gracefully shuts down the agent
func (a *Agent) Stop() error {
	a.cancel()
	if a.listener != nil {
		a.listener.Close()
	}
	a.wg.Wait()
	return nil
}

// EnsureNetwork creates the panel_apps network if it doesn't exist
func (a *Agent) EnsureNetwork(ctx context.Context) error {
	return a.network.EnsureNetwork(ctx)
}

// HealthCheckParams holds parameters for starting a health check
type HealthCheckParams struct {
	AppID       string
	ContainerID string
	Port        int
	Path        string
	Interval    int
	Timeout     int
	Retries     int
}

// StartHealthCheck begins monitoring container health
func (a *Agent) StartHealthCheck(ctx context.Context, params HealthCheckParams) {
	a.health.Start(ctx, params)
}

// StopHealthCheck stops monitoring a container's health
func (a *Agent) StopHealthCheck(appID string) {
	a.health.Stop(appID)
}

// BuildParams holds parameters for building an image
type BuildParams struct {
	AppID         string `json:"app_id"`
	AppName       string `json:"app_name"`
	RepoURL       string `json:"repo_url"`
	Branch        string `json:"branch"`
	Commit        string `json:"commit"`
	BuildStrategy string `json:"build_strategy"` // nixpacks, dockerfile, static
	BuildCommand  string `json:"build_command"`
	StartCommand  string `json:"start_command"`
	BuildPath     string `json:"build_path"` // repo clone destination
}

// BuildResult holds the result of a build operation
type BuildResult struct {
	ImageName  string `json:"image_name"`
	CommitSHA  string `json:"commit_sha"`
	Duration   int    `json:"duration_seconds"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	BuildLogs  []LogLine `json:"build_logs,omitempty"`
}

// LogLine represents a single log entry
type LogLine struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"` // info, warn, error
	Message   string `json:"message"`
	Stream    string `json:"stream,omitempty"` // stdout, stderr
}

// RunParams holds parameters for running a container
type RunParams struct {
	AppID       string            `json:"app_id"`
	Image       string            `json:"image"`
	EnvVars     map[string]string `json:"env_vars"`
	Volumes     []VolumeMount     `json:"volumes"`
	Ports       []PortMapping     `json:"ports"`
	Network     string            `json:"network"`
	Restart     string            `json:"restart_policy"` // always, unless-stopped, no
	MemoryLimit string            `json:"memory_limit"`    // e.g., "512m"
	CPUQuota    int64             `json:"cpu_quota"`       // CPU quota in microseconds
}

// VolumeMount represents a volume mount
type VolumeMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only"`
}

// PortMapping represents port mapping
type PortMapping struct {
	Internal int `json:"internal"`
	External int `json:"external"` // 0 = auto-assign
}

// RunResult holds the result of creating/running a container
type RunResult struct {
	ContainerID string `json:"container_id"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	Port        int    `json:"port"`
}

// StopParams holds parameters for stopping a container
type StopParams struct {
	ContainerID string `json:"container_id"`
	Timeout     int    `json:"timeout"` // seconds, 0 = default 10
}

// StartParams holds parameters for starting a container
type StartParams struct {
	ContainerID string `json:"container_id"`
}

// RestartParams holds parameters for restarting a container
type RestartParams struct {
	ContainerID string `json:"container_id"`
}

// LogsParams holds parameters for getting container logs
type LogsParams struct {
	ContainerID string `json:"container_id"`
	Stream      string `json:"stream"` // stdout, stderr, both
	Tail        int    `json:"tail"`    // number of lines, 0 = all
}

// ExecParams holds parameters for executing a command in a container
type ExecParams struct {
	ContainerID string   `json:"container_id"`
	Command     []string `json:"command"` // command and arguments
	WorkDir     string   `json:"work_dir"`
}

// ExecResult holds the result of executing a command
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
	Error    string `json:"error,omitempty"`
}

// StatsParams holds parameters for getting container stats
type StatsParams struct {
	ContainerID string `json:"container_id"`
}

// ContainerStats holds container resource usage
type ContainerStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsageMB int     `json:"memory_usage_mb"`
	MemoryLimitMB int     `json:"memory_limit_mb"`
	NetworkRX     int64   `json:"network_rx_bytes"`
	NetworkTX     int64   `json:"network_tx_bytes"`
}

// RemoveParams holds parameters for removing a container
type RemoveParams struct {
	ContainerID string `json:"container_id"`
	Force       bool   `json:"force"`
}

// Client provides a client for communicating with the agent
type Client struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
	useTCP     bool
	tcpAddress string
}

// NewClient creates a new agent client.
// It defaults to the Unix socket path and falls back to TCP on the default agent port.
// The caller should call Connect() before issuing commands to pre-warm the connection.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}
	return &Client{
		socketPath:  socketPath,
		useTCP:     false,
		tcpAddress: fmt.Sprintf("127.0.0.1:%d", DefaultAgentTCPPort),
	}
}

// Connect establishes connection to the agent
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked()
}

// ConnectContext establishes connection to the agent with a context timeout.
func (c *Client) ConnectContext(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Unix socket dial with context deadline
	unixDialer := net.Dialer{Timeout: 5 * time.Second }
	conn, err := unixDialer.DialContext(ctx, "unix", c.socketPath)
	if err == nil {
		c.conn = conn
		c.useTCP = false
		return nil
	}

	// Fallback to TCP
	tcpDialer := net.Dialer{Timeout: 5 * time.Second }
	conn, err = tcpDialer.DialContext(ctx, "tcp", c.tcpAddress)
	if err != nil {
		return fmt.Errorf("agent unreachable (unix: %s, tcp: %s): %w", c.socketPath, c.tcpAddress, err)
	}
	c.conn = conn
	c.useTCP = true
	return nil
}

// Close closes the client connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// sendCommand sends a command and waits for response.
// If the connection is dead, it reconnects once and retries before returning an error.
func (c *Client) sendCommand(cmdType string, params interface{}) (Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ensureConnected := func() error {
		if c.conn != nil {
			return nil
		}
		return c.connectLocked()
	}

	if err := ensureConnected(); err != nil {
		return Response{}, fmt.Errorf("not connected (tried unix socket %s and tcp %s: %w)", c.socketPath, c.tcpAddress, err)
	}

	cmd := Command{
		Type:      cmdType,
		RequestID: generateRequestID(),
		Params:    mustMarshal(params),
	}

	// Helper to encode and decode with a live connection check
	send := func() (Response, error) {
		if err := json.NewEncoder(c.conn).Encode(cmd); err != nil {
			return Response{}, fmt.Errorf("failed to send command: %w", err)
		}
		var resp Response
		if err := json.NewDecoder(c.conn).Decode(&resp); err != nil {
			return Response{}, fmt.Errorf("failed to read response: %w", err)
		}
		return resp, nil
	}

	// Try once; if it fails due to a closed connection, reconnect and retry once.
	resp, err := send()
	if err != nil {
		// Check if it's a "use of closed network connection" or similar
		if isConnClosedErr(err) {
			if reconnErr := c.connectLocked(); reconnErr == nil {
				resp, err = send()
			}
		}
		if err != nil {
			// Don't leave a broken connection around
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			return Response{}, fmt.Errorf("agent command %s failed after reconnect: %w", cmdType, err)
		}
	}

	return resp, nil
}

// isConnClosedErr returns true if the error indicates the connection was closed.
func isConnClosedErr(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "use of closed") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "i/o timeout") ||
		strings.Contains(errStr, "EOF")
}

// connectLocked establishes a new connection. Caller must hold c.mu.
func (c *Client) connectLocked() error {
	// Try Unix socket first
	conn, err := net.DialTimeout("unix", c.socketPath, 5*time.Second)
	if err == nil {
		c.conn = conn
		c.useTCP = false
		return nil
	}

	// Fallback to TCP
	conn, err = net.DialTimeout("tcp", c.tcpAddress, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to agent (tried unix socket %s and tcp %s: %w)", c.socketPath, c.tcpAddress, err)
	}
	c.conn = conn
	c.useTCP = true
	return nil
}

// Ping checks agent connectivity
func (c *Client) Ping() error {
	resp, err := c.sendCommand(CmdPing, nil)
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("agent error: %s", resp.Error)
	}
	return nil
}

// Build triggers a Docker image build
func (c *Client) Build(ctx context.Context, params BuildParams) (*BuildResult, error) {
	resp, err := c.sendCommand(CmdBuild, params)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("build failed: %s", resp.Error)
	}
	result := &BuildResult{}
	if err := unmarshalResponse(resp.Data, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Run creates and starts a container
func (c *Client) Run(ctx context.Context, params RunParams) (*RunResult, error) {
	resp, err := c.sendCommand(CmdRun, params)
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("run failed: %s", resp.Error)
	}
	result := &RunResult{}
	if err := unmarshalResponse(resp.Data, result); err != nil {
		return nil, err
	}
	return result, nil
}

// Stop stops a container
func (c *Client) Stop(ctx context.Context, containerID string, timeout int) error {
	resp, err := c.sendCommand(CmdStop, StopParams{ContainerID: containerID, Timeout: timeout})
	if err != nil {
		return err
	}
	if resp.Status != "success" {
		return fmt.Errorf("stop failed: %s", resp.Error)
	}
	return nil
}

// Start starts a container
func (c *Client) Start(ctx context.Context, containerID string) error {
	resp, err := c.sendCommand(CmdStart, StartParams{ContainerID: containerID})
	if err != nil {
		return err
	}
	if resp.Status != "success" {
		return fmt.Errorf("start failed: %s", resp.Error)
	}
	return nil
}

// Restart restarts a container
func (c *Client) Restart(ctx context.Context, containerID string) error {
	resp, err := c.sendCommand(CmdRestart, RestartParams{ContainerID: containerID})
	if err != nil {
		return err
	}
	if resp.Status != "success" {
		return fmt.Errorf("restart failed: %s", resp.Error)
	}
	return nil
}

// GetLogs retrieves container logs
func (c *Client) GetLogs(ctx context.Context, containerID string, stream string, tail int) ([]LogLine, error) {
	resp, err := c.sendCommand(CmdLogs, LogsParams{ContainerID: containerID, Stream: stream, Tail: tail})
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("logs failed: %s", resp.Error)
	}
	var logs []LogLine
	if err := unmarshalResponse(resp.Data, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// Exec executes a command in a container
func (c *Client) Exec(ctx context.Context, containerID string, command []string, workDir string) (*ExecResult, error) {
	resp, err := c.sendCommand(CmdExec, ExecParams{ContainerID: containerID, Command: command, WorkDir: workDir})
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("exec failed: %s", resp.Error)
	}
	result := &ExecResult{}
	if err := unmarshalResponse(resp.Data, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStats retrieves container stats
func (c *Client) GetStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	resp, err := c.sendCommand(CmdStats, StatsParams{ContainerID: containerID})
	if err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("stats failed: %s", resp.Error)
	}
	stats := &ContainerStats{}
	if err := unmarshalResponse(resp.Data, stats); err != nil {
		return nil, err
	}
	return stats, nil
}

// Remove removes a container
func (c *Client) Remove(ctx context.Context, containerID string, force bool) error {
	resp, err := c.sendCommand(CmdRemove, RemoveParams{ContainerID: containerID, Force: force})
	if err != nil {
		return err
	}
	if resp.Status != "success" {
		return fmt.Errorf("remove failed: %s", resp.Error)
	}
	return nil
}

// StartHealthCheck starts monitoring a container's health
func (c *Client) StartHealthCheck(ctx context.Context, params HealthCheckParams) error {
	resp, err := c.sendCommand(CmdHealthCheck, params)
	if err != nil {
		return err
	}
	if resp.Status != "success" {
		return fmt.Errorf("health check failed: %s", resp.Error)
	}
	return nil
}

// Helper functions
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		log.Fatalf("Failed to marshal: %v", err)
	}
	return data
}

func unmarshalResponse(data interface{}, v interface{}) error {
	if data == nil {
		return nil
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, v)
}

// SanitizeAppName sanitizes app name for use in image names
func SanitizeAppName(name string) string {
	// Replace invalid characters with hyphens
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")
	// Remove any characters that aren't alphanumeric or hyphens
	var result strings.Builder
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			result.WriteRune(c)
		}
	}
	return strings.ToLower(result.String())
}

// GetBuildContextDir returns the build context directory for an app
func GetBuildContextDir(dataDir, appID, deploymentID string) string {
	return filepath.Join(dataDir, "tmp", "builds", deploymentID)
}

// GetRepoDir returns the repository directory for an app
func GetRepoDir(dataDir, appID string) string {
	return filepath.Join(dataDir, "tmp", appID, "repo")
}
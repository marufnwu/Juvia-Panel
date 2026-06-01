package agent

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HealthChecker monitors container health via HTTP health checks
type HealthChecker struct {
	checks       map[string]*healthMonitor
	mu           sync.RWMutex
	containerMgr *ContainerManager
}

// healthMonitor holds health check state for a container
type healthMonitor struct {
	AppID       string
	ContainerID string
	Port        int
	Path        string
	Interval    int
	Timeout     int
	Retries     int
	FailCount   int
	stopChan    chan struct{}
}

// NewHealthChecker creates a new HealthChecker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks:       make(map[string]*healthMonitor),
		containerMgr: NewContainerManager(),
	}
}

// Start begins monitoring a container's health
func (hc *HealthChecker) Start(ctx context.Context, params HealthCheckParams) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Stop existing health check if any
	if existing, ok := hc.checks[params.AppID]; ok {
		close(existing.stopChan)
	}

	monitor := &healthMonitor{
		AppID:       params.AppID,
		ContainerID: params.ContainerID,
		Port:        params.Port,
		Path:        params.Path,
		Interval:    params.Interval,
		Timeout:     params.Timeout,
		Retries:     params.Retries,
		FailCount:   0,
		stopChan:    make(chan struct{}),
	}

	hc.checks[params.AppID] = monitor

	// Start monitoring goroutine
	go hc.monitor(monitor)
}

// Stop stops monitoring a container's health
func (hc *HealthChecker) Stop(appID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if monitor, ok := hc.checks[appID]; ok {
		close(monitor.stopChan)
		delete(hc.checks, appID)
	}
}

// monitor runs the health check loop
func (hc *HealthChecker) monitor(m *healthMonitor) {
	// Set defaults
	if m.Interval <= 0 {
		m.Interval = 30 // default 30 seconds
	}
	if m.Timeout <= 0 {
		m.Timeout = 5 // default 5 seconds
	}
	if m.Retries <= 0 {
		m.Retries = 3 // default 3 retries
	}

	// Add path prefix if not present
	path := m.Path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	ticker := time.NewTicker(time.Duration(m.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			healthy := hc.checkHealth(m.Port, path, m.Timeout)

			if healthy {
				m.FailCount = 0
			} else {
				m.FailCount++
				fmt.Printf("Health check failed for app %s (attempt %d/%d)\n", m.AppID, m.FailCount, m.Retries)

				if m.FailCount >= m.Retries {
					fmt.Printf("Health check failed %d times for app %s, restarting container\n", m.FailCount, m.AppID)
					hc.restartContainer(m)
					m.FailCount = 0 // Reset after restart
				}
			}
		}
	}
}

// checkHealth performs a single health check
func (hc *HealthChecker) checkHealth(port int, path string, timeout int) bool {
	url := fmt.Sprintf("http://localhost:%d%s", port, path)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Consider 2xx and 3xx as healthy
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// restartContainer restarts a container and updates health check parameters
func (hc *HealthChecker) restartContainer(m *healthMonitor) {
	ctx := context.Background()

	// Get current container info to find new container ID if it changed
	info, err := hc.containerMgr.GetContainerInfo(ctx, m.ContainerID)
	if err != nil {
		fmt.Printf("Failed to get container info: %v\n", err)
		return
	}

	// Restart the container
	if err := hc.containerMgr.RestartContainer(ctx, m.ContainerID); err != nil {
		fmt.Printf("Failed to restart container: %v\n", err)
		return
	}

	// Get updated port if auto-assigned
	if newPort, ok := info["Port"]; ok {
		if portMap, ok := newPort.(map[string]interface{}); ok {
			if hostPort, ok := portMap["8080/tcp"]; ok {
				// Update port for future health checks
				if hp, ok := hostPort.([]interface{}); ok && len(hp) > 0 {
					if portDict, ok := hp[0].(map[string]interface{}); ok {
						if hostPortVal, ok := portDict["HostPort"]; ok {
							if portStr, ok := hostPortVal.(string); ok {
								var newPortVal int
								fmt.Sscanf(portStr, "%d", &newPortVal)
								if newPortVal > 0 {
									m.Port = newPortVal
								}
							}
						}
					}
				}
			}
		}
	}

	fmt.Printf("Container %s restarted successfully\n", m.ContainerID)
}

// GetHealthStatus returns the current health status of a container
func (hc *HealthChecker) GetHealthStatus(appID string) (bool, int) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if monitor, ok := hc.checks[appID]; ok {
		return monitor.FailCount == 0, monitor.FailCount
	}

	return false, 0
}

// IsMonitored checks if an app is being monitored
func (hc *HealthChecker) IsMonitored(appID string) bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	_, ok := hc.checks[appID]
	return ok
}

// GetMonitoredApps returns list of apps being monitored
func (hc *HealthChecker) GetMonitoredApps() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var apps []string
	for appID := range hc.checks {
		apps = append(apps, appID)
	}
	return apps
}

// StopAll stops all health checks
func (hc *HealthChecker) StopAll() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	for _, monitor := range hc.checks {
		close(monitor.stopChan)
	}
	hc.checks = make(map[string]*healthMonitor)
}
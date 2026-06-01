package agent

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
)

// NetworkManager handles Docker network operations
type NetworkManager struct {
	networkName string
	driver      string
}

// NewNetworkManager creates a new NetworkManager
func NewNetworkManager() *NetworkManager {
	return &NetworkManager{
		networkName: "panel_apps",
		driver:      "bridge",
	}
}

// SetNetworkName sets the Docker network name
func (nm *NetworkManager) SetNetworkName(name string) {
	nm.networkName = name
}

// EnsureNetwork creates the panel_apps Docker network if it doesn't exist
func (nm *NetworkManager) EnsureNetwork(ctx context.Context) error {
	// Check if network exists
	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--format", "{{.Name}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		if scanner.Text() == nm.networkName {
			return nil // network already exists
		}
	}

	// Create the network
	createCmd := exec.CommandContext(ctx, "docker", "network", "create", "--driver", nm.driver, nm.networkName)
	_, err = createCmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

// RemoveNetwork removes the panel_apps network
func (nm *NetworkManager) RemoveNetwork(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "network", "rm", nm.networkName)
	_, err := cmd.CombinedOutput()
	return err
}

// NetworkExists checks if the network exists
func (nm *NetworkManager) NetworkExists(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "network", "ls", "--format", "{{.Name}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		if scanner.Text() == nm.networkName {
			return true, nil
		}
	}

	return false, nil
}

// GetNetworkInfo returns information about the network
func (nm *NetworkManager) GetNetworkInfo(ctx context.Context) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", nm.networkName,
		"--format", "{{.Id}}:{{.Driver}}:{{.Scope}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	info := make(map[string]string)
	parts := strings.Split(strings.TrimSpace(string(output)), ":")
	if len(parts) >= 3 {
		info["id"] = parts[0]
		info["driver"] = parts[1]
		info["scope"] = parts[2]
	}

	return info, nil
}

// ListContainersInNetwork returns all containers connected to the network
func (nm *NetworkManager) ListContainersInNetwork(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", nm.networkName,
		"--format", "{{range .Containers}}{{.Name}} {{end}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var containers []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name != "" {
			containers = append(containers, name)
		}
	}

	return containers, nil
}

// ConnectContainer connects a container to the network
func (nm *NetworkManager) ConnectContainer(ctx context.Context, containerID, containerName string) error {
	cmd := exec.CommandContext(ctx, "docker", "network", "connect", nm.networkName, containerName)
	_, err := cmd.CombinedOutput()
	return err
}

// DisconnectContainer disconnects a container from the network
func (nm *NetworkManager) DisconnectContainer(ctx context.Context, containerName string) error {
	cmd := exec.CommandContext(ctx, "docker", "network", "disconnect", nm.networkName, containerName)
	_, err := cmd.CombinedOutput()
	return err
}
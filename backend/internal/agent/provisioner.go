package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Provisioner handles database service provisioning
type Provisioner struct {
	containerManager *ContainerManager
	dataDir          string
	networkName      string
}

// ServiceInfo holds information about a provisioned service
type ServiceInfo struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Version          string            `json:"version"`
	Status           string            `json:"status"`
	Port             int               `json:"port"`
	ContainerID      string            `json:"container_id"`
	ContainerImage   string            `json:"container_image"`
	ConnectionString string            `json:"connection_string"` // masked
	Credentials      ServiceCredentials `json:"credentials"`
}

// ServiceCredentials holds service connection credentials
type ServiceCredentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"` // masked in responses
}

// NewProvisioner creates a new Provisioner
func NewProvisioner() *Provisioner {
	return &Provisioner{
		containerManager: NewContainerManager(),
		dataDir:          "/var/panel",
		networkName:      "panel_apps",
	}
}

// SetDataDir sets the data directory
func (p *Provisioner) SetDataDir(dir string) {
	p.dataDir = dir
	p.containerManager.SetDataDir(dir)
}

// SetNetworkName sets the Docker network name
func (p *Provisioner) SetNetworkName(name string) {
	p.networkName = name
	p.containerManager.SetNetworkName(name)
}

// ProvisionPostgreSQL provisions a PostgreSQL service
func (p *Provisioner) ProvisionPostgreSQL(ctx context.Context, id, password string) (*ServiceInfo, error) {
	envVars := map[string]string{
		"POSTGRES_PASSWORD": password,
		"POSTGRES_DB":      id,
	}
	return p.provisionService(ctx, id, "postgresql", "postgres:16-alpine", 5432, password, envVars, "/var/lib/postgresql/data")
}

// ProvisionRedis provisions a Redis service
func (p *Provisioner) ProvisionRedis(ctx context.Context, id, password string) (*ServiceInfo, error) {
	// Redis doesn't require password for basic AUTH but we support it
	envVars := map[string]string{}
	if password != "" {
		envVars["REDIS_PASSWORD"] = password
	}
	return p.provisionService(ctx, id, "redis", "redis:7-alpine", 6379, password, envVars, "/data")
}

// ProvisionMySQL provisions a MySQL service
func (p *Provisioner) ProvisionMySQL(ctx context.Context, id, password string) (*ServiceInfo, error) {
	envVars := map[string]string{
		"MYSQL_ROOT_PASSWORD": password,
		"MYSQL_DATABASE":      id,
	}
	return p.provisionService(ctx, id, "mysql", "mysql:8", 3306, password, envVars, "/var/lib/mysql")
}

// ProvisionMongoDB provisions a MongoDB service
func (p *Provisioner) ProvisionMongoDB(ctx context.Context, id, password string) (*ServiceInfo, error) {
	envVars := map[string]string{
		"MONGO_INITDB_ROOT_USERNAME": id,
		"MONGO_INITDB_ROOT_PASSWORD": password,
	}
	return p.provisionService(ctx, id, "mongodb", "mongo:7", 27017, password, envVars, "/data/db")
}

// provisionService is a helper that provisions a database service container
func (p *Provisioner) provisionService(ctx context.Context, id, serviceType, image string, defaultPort int, password string, envVars map[string]string, volumePath string) (*ServiceInfo, error) {
	provisionCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Create data directory
	dataPath := fmt.Sprintf("%s/services/%s/data", p.dataDir, id)
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Determine port - use default port for services
	port := defaultPort

	// Build volume mount
	volumes := []VolumeMount{
		{
			HostPath:      dataPath,
			ContainerPath: volumePath,
			ReadOnly:      false,
		},
	}

	// Determine internal port based on service type
	internalPort := getInternalPort(serviceType)

	// Run the container
	result, err := p.containerManager.RunContainer(
		provisionCtx,
		image,
		fmt.Sprintf("panel-svc-%s", id),
		envVars,
		volumes,
		[]PortMapping{{Internal: internalPort, External: port}},
		p.networkName,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Wait for container to be healthy (for databases)
	if err := p.waitForHealthy(provisionCtx, result.ContainerID, 60*time.Second); err != nil {
		// Log warning but don't fail - container might still be starting
		fmt.Printf("Warning: container %s may not be healthy yet: %v\n", result.ContainerID, err)
	}

	// Build service info
	info := &ServiceInfo{
		ID:             id,
		Name:           id,
		Type:           serviceType,
		Version:        getVersionFromImage(image),
		Status:         "running",
		Port:           port,
		ContainerID:    result.ContainerID,
		ContainerImage: image,
	}

	// Set credentials based on service type
	info.Credentials = buildCredentials(serviceType, id, password, port)
	info.ConnectionString = buildConnectionString(serviceType, id, password, port)

	return info, nil
}

// waitForHealthy waits for a container to become healthy
func (p *Provisioner) waitForHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(timeout)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check container health status
			cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Health.Status}}", containerID)
			output, err := cmd.CombinedOutput()
			if err != nil {
				// Container might not have health check - try basic status
				cmd = exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", containerID)
				output, err := cmd.CombinedOutput()
				if err == nil {
					status := strings.TrimSpace(string(output))
					if status == "running" {
						return nil
					}
				}
				continue
			}

			healthStatus := strings.TrimSpace(string(output))
			if healthStatus == "healthy" {
				return nil
			}

			// Check timeout
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for container to be healthy")
			}
		}
	}
}

// StopService stops a service container
func (p *Provisioner) StopService(ctx context.Context, serviceID string) error {
	containerName := fmt.Sprintf("panel-svc-%s", serviceID)
	cmd := exec.CommandContext(ctx, "docker", "stop", "-t", "10", containerName)
	_, err := cmd.CombinedOutput()
	return err
}

// StartService starts a service container
func (p *Provisioner) StartService(ctx context.Context, serviceID string) error {
	containerName := fmt.Sprintf("panel-svc-%s", serviceID)
	cmd := exec.CommandContext(ctx, "docker", "start", containerName)
	_, err := cmd.CombinedOutput()
	return err
}

// RestartService restarts a service container
func (p *Provisioner) RestartService(ctx context.Context, serviceID string) error {
	containerName := fmt.Sprintf("panel-svc-%s", serviceID)
	cmd := exec.CommandContext(ctx, "docker", "restart", containerName)
	_, err := cmd.CombinedOutput()
	return err
}

// RemoveService removes a service container
func (p *Provisioner) RemoveService(ctx context.Context, serviceID string, force bool) error {
	containerName := fmt.Sprintf("panel-svc-%s", serviceID)
	args := []string{"rm"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, containerName)

	cmd := exec.CommandContext(ctx, "docker", args...)
	_, err := cmd.CombinedOutput()
	return err
}

// GetServiceStatus returns the status of a service container
func (p *Provisioner) GetServiceStatus(ctx context.Context, serviceID string) (string, error) {
	containerName := fmt.Sprintf("panel-svc-%s", serviceID)
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", containerName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetServiceLogs returns logs from a service container
func (p *Provisioner) GetServiceLogs(ctx context.Context, serviceID string, tail int) ([]LogLine, error) {
	containerName := fmt.Sprintf("panel-svc-%s", serviceID)
	args := []string{"logs", "--tail", strconv.Itoa(tail), "-t", containerName}
	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	return parseDockerLogs(string(output)), nil
}

// getInternalPort returns the internal port for a service type
func getInternalPort(serviceType string) int {
	ports := map[string]int{
		"postgresql": 5432,
		"mysql":      3306,
		"mongodb":    27017,
		"redis":      6379,
	}
	if port, ok := ports[serviceType]; ok {
		return port
	}
	return 5432
}

// getVersionFromImage extracts version from docker image name
func getVersionFromImage(image string) string {
	parts := strings.Split(image, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "latest"
}

// buildCredentials builds service credentials
func buildCredentials(serviceType, id, password string, port int) ServiceCredentials {
	creds := ServiceCredentials{
		Host: "localhost",
		Port: port,
	}

	switch serviceType {
	case "postgresql":
		creds.Database = id
		creds.Username = id + "-user"
	case "mysql":
		creds.Database = id
		creds.Username = id + "-user"
	case "mongodb":
		creds.Database = "admin"
		creds.Username = id
	case "redis":
		creds.Database = "0"
		creds.Username = "default"
	}

	// Mask password
	creds.Password = maskPassword(password)

	return creds
}

// buildConnectionString builds a connection string for the service
func buildConnectionString(serviceType, id, password string, port int) string {
	switch serviceType {
	case "postgresql":
		return fmt.Sprintf("postgres://%s-user:****@localhost:%d/%s", id, port, id)
	case "mysql":
		return fmt.Sprintf("mysql://%s-user:****@localhost:%d/%s", id, port, id)
	case "mongodb":
		return fmt.Sprintf("mongodb://%s:****@localhost:%d/admin", id, port)
	case "redis":
		return fmt.Sprintf("redis://localhost:%d/0", port)
	default:
		return fmt.Sprintf("%s://localhost:%d", serviceType, port)
	}
}

// maskPassword masks a password for display
func maskPassword(password string) string {
	if len(password) <= 8 {
		return "********"
	}
	return password[:4] + "..." + password[len(password)-4:]
}

// generatePassword generates a random password
func generatePassword() string {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
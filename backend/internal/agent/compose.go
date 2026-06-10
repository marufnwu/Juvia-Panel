package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type ComposeManager struct{}

func NewComposeManager() *ComposeManager {
	return &ComposeManager{}
}

type ComposeUpParams struct {
	ProjectName   string `json:"project_name"`
	ComposeFile   string `json:"compose_file"`
	EnvFile       string `json:"env_file,omitempty"`
	Detach        bool   `json:"detach"`
}

type ComposeDownParams struct {
	ProjectName      string `json:"project_name"`
	ComposeFile      string `json:"compose_file"`
	RemoveVolumes    bool   `json:"remove_volumes"`
	RemoveImages     string `json:"remove_images,omitempty"`
}

type ComposeStopParams struct {
	ProjectName string `json:"project_name"`
	ComposeFile string `json:"compose_file"`
}

type ComposeStartParams struct {
	ProjectName string `json:"project_name"`
	ComposeFile string `json:"compose_file"`
}

type ComposeRestartParams struct {
	ProjectName string `json:"project_name"`
	ComposeFile string `json:"compose_file"`
}

type ComposeLogsParams struct {
	ProjectName string `json:"project_name"`
	ComposeFile string `json:"compose_file"`
	Service     string `json:"service,omitempty"`
	Stream      string `json:"stream,omitempty"`
	Tail        int    `json:"tail"`
}

type ComposePsParams struct {
	ProjectName string `json:"project_name"`
	ComposeFile string `json:"compose_file"`
}

type ComposeExecParams struct {
	ProjectName string `json:"project_name"`
	ComposeFile string `json:"compose_file"`
	Service     string `json:"service"`
	Command     string `json:"command"`
	WorkDir     string `json:"work_dir,omitempty"`
}

type ComposeResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Service   string `json:"service,omitempty"`
	Project   string `json:"project,omitempty"`
	Services  []string `json:"services,omitempty"`
}

type ComposeServiceInfo struct {
	Name       string `json:"name"`
	Image      string `json:"image"`
	State      string `json:"state"`
	Ports      string `json:"ports"`
	ContainerID string `json:"container_id"`
	Status     string `json:"status"`
}

func (m *ComposeManager) composeCmd(ctx context.Context, projectName, composeFile string, args ...string) (*exec.Cmd, error) {
	if composeFile == "" {
		composeFile = "docker-compose.yml"
	}
	baseArgs := []string{
		"-p", projectName,
		"-f", composeFile,
	}
	return exec.CommandContext(ctx, "docker", append(baseArgs, args...)...), nil
}

func (m *ComposeManager) ComposeUp(ctx context.Context, params ComposeUpParams) (*ComposeResult, error) {
	args := []string{"up", "-d"}
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, args...)
	if err != nil {
		return nil, err
	}

	if params.EnvFile != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("ENV_FILE=%s", params.EnvFile))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ComposeResult{
			Success: false,
			Message: string(output),
		}, fmt.Errorf("compose up failed: %w", err)
	}

	services, _ := m.ComposeServices(ctx, params.ProjectName, params.ComposeFile)

	return &ComposeResult{
		Success:  true,
		Message:  "compose up succeeded",
		Project:  params.ProjectName,
		Services: services,
	}, nil
}

func (m *ComposeManager) ComposeDown(ctx context.Context, params ComposeDownParams) (*ComposeResult, error) {
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	args := []string{"down"}
	if params.RemoveVolumes {
		args = append(args, "-v")
	}
	if params.RemoveImages != "" {
		args = append(args, "--rmi", params.RemoveImages)
	}

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, args...)
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ComposeResult{
			Success: false,
			Message: string(output),
		}, fmt.Errorf("compose down failed: %w", err)
	}

	return &ComposeResult{
		Success: true,
		Message: "compose down succeeded",
		Project: params.ProjectName,
	}, nil
}

func (m *ComposeManager) ComposeStop(ctx context.Context, params ComposeStopParams) (*ComposeResult, error) {
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, "stop")
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ComposeResult{
			Success: false,
			Message: string(output),
		}, fmt.Errorf("compose stop failed: %w", err)
	}

	return &ComposeResult{
		Success:  true,
		Message:  "compose stop succeeded",
		Project:  params.ProjectName,
	}, nil
}

func (m *ComposeManager) ComposeStart(ctx context.Context, params ComposeStartParams) (*ComposeResult, error) {
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, "start")
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ComposeResult{
			Success: false,
			Message: string(output),
		}, fmt.Errorf("compose start failed: %w", err)
	}

	return &ComposeResult{
		Success:  true,
		Message:  "compose start succeeded",
		Project:  params.ProjectName,
	}, nil
}

func (m *ComposeManager) ComposeRestart(ctx context.Context, params ComposeRestartParams) (*ComposeResult, error) {
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, "restart")
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ComposeResult{
			Success: false,
			Message: string(output),
		}, fmt.Errorf("compose restart failed: %w", err)
	}

	return &ComposeResult{
		Success:  true,
		Message:  "compose restart succeeded",
		Project:  params.ProjectName,
	}, nil
}

func (m *ComposeManager) ComposeLogs(ctx context.Context, params ComposeLogsParams) ([]LogLine, error) {
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	args := []string{"logs"}
	if params.Service != "" {
		args = append(args, params.Service)
	}
	if params.Tail > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", params.Tail))
	} else {
		args = append(args, "--tail", "100")
	}
	if params.Stream == "stdout" || params.Stream == "stderr" {
		args = append(args, "--follow")
	}

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, args...)
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compose logs failed: %w", err)
	}

	var logs []LogLine
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			logs = append(logs, LogLine{
				Timestamp: "",
				Level:     "info",
				Message:   line,
			})
		}
	}

	return logs, nil
}

func (m *ComposeManager) ComposePs(ctx context.Context, params ComposePsParams) ([]ComposeServiceInfo, error) {
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, "ps", "--format", "json")
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compose ps failed: %w", err)
	}

	var services []ComposeServiceInfo

	if strings.TrimSpace(string(output)) == "" {
		return services, nil
	}

	var rawOutput []map[string]interface{}
	if err := json.Unmarshal(output, &rawOutput); err != nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if line = strings.TrimSpace(line); line == "" {
				continue
			}
			var svc map[string]interface{}
			if err := json.Unmarshal([]byte(line), &svc); err == nil {
				rawOutput = append(rawOutput, svc)
			}
		}
	}

	for _, svc := range rawOutput {
		name, _ := svc["Service"].(string)
		if name == "" {
			continue
		}
		info := ComposeServiceInfo{
			Name: name,
		}
		if v, ok := svc["Image"].(string); ok {
			info.Image = v
		}
		if v, ok := svc["State"].(string); ok {
			info.State = v
		}
		if v, ok := svc["Ports"].(string); ok {
			info.Ports = v
		}
		if v, ok := svc["Container"].(string); ok {
			info.ContainerID = v
		}
		if v, ok := svc["Status"].(string); ok {
			info.Status = v
		}
		services = append(services, info)
	}

	return services, nil
}

func (m *ComposeManager) ComposeServices(ctx context.Context, projectName, composeFile string) ([]string, error) {
	if composeFile == "" {
		composeFile = "docker-compose.yml"
	}

	cmd, err := m.composeCmd(ctx, projectName, composeFile, "config", "--services")
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("compose config --services failed: %w", err)
	}

	var services []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			services = append(services, line)
		}
	}
	return services, nil
}

func (m *ComposeManager) ComposeExec(ctx context.Context, params ComposeExecParams) (*ExecResult, error) {
	if params.ComposeFile == "" {
		params.ComposeFile = "docker-compose.yml"
	}

	args := []string{"exec", "-d"}
	if params.WorkDir != "" {
		args = append(args, "-w", params.WorkDir)
	}
	args = append(args, params.Service)
	args = append(args, strings.Fields(params.Command)...)

	cmd, err := m.composeCmd(ctx, params.ProjectName, params.ComposeFile, args...)
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &ExecResult{
			Output:   string(output),
			ExitCode: 1,
		}, fmt.Errorf("compose exec failed: %w", err)
	}

	return &ExecResult{
		Output:   string(output),
		ExitCode: 0,
	}, nil
}
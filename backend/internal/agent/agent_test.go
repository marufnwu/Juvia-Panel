package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeAppName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "myapp", "myapp"},
		{"with hyphens", "my-app", "my-app"},
		{"with underscores", "my_app", "my-app"},
		{"with dots", "my.app", "myapp"},
		{"with spaces", "my app", "my-app"},
		{"with special chars", "my@app!#", "myapp"},
		{"uppercase", "MyApp", "myapp"},
		{"mixed chars", "My_App.Name!", "my-appname"},
		{"empty", "", ""},
		{"numbers only", "123", "123"},
		{"long name with digits", "app-v2.1.0_test", "app-v210-test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeAppName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBuildContextDir(t *testing.T) {
	dir := GetBuildContextDir("/var/panel", "app_abc123", "deadbeef1234")
	assert.Contains(t, dir, "var")
	assert.Contains(t, dir, "panel")
	assert.Contains(t, dir, "deadbeef1234")

	dir = GetBuildContextDir("/data", "test", "commit")
	assert.Contains(t, dir, "data")
	assert.Contains(t, dir, "commit")
}

func TestParseMemoryToMB(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
		wantErr  bool
	}{
		{"plain MB", "512", 512, false},
		{"MiB suffix", "256MiB", 256, false},
		{"MB suffix", "256MB", 256, false},
		{"GiB suffix", "1GiB", 1024, false},
		{"GB suffix", "2GB", 2048, false},
		{"lowercase", "256mib", 256, false},
		{"uppercase", "1GIB", 1024, false},
		{"with space", "  512  ", 512, false},
		{"decimal MB", "1.5", 1, false}, // truncated to int
		{"GB decimal", "1.5GB", 1536, false},
		{"invalid format", "abc", 0, true},
		{"zero", "0", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMemoryToMB(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseDockerLogs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // number of log lines
	}{
		{
			"single line",
			"2024-06-01T12:34:56Z Hello World",
			1,
		},
		{
			"multiple lines",
			"2024-06-01T12:34:56Z line1\n2024-06-01T12:34:57Z line2",
			2,
		},
		{
			"with stdout prefix",
			"2024-06-01T12:34:56Z stdout Server started",
			1,
		},
		{
			"with stderr prefix",
			"2024-06-01T12:34:56Z stderr Error occurred",
			1,
		},
		{
			"empty",
			"",
			0,
		},
		{
			"line without space",
			"2024-06-01T12:34:56Z",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := parseDockerLogs(tt.input)
			assert.Len(t, logs, tt.expected)

			for _, log := range logs {
				assert.NotEmpty(t, log.Timestamp)
				assert.Equal(t, "info", log.Level)
				assert.NotEmpty(t, log.Message)
			}
		})
	}
}

func TestParseDockerLogs_StreamDetection(t *testing.T) {
	input := "2024-06-01T12:34:56Z stdout Log message"
	logs := parseDockerLogs(input)
	assert.Len(t, logs, 1)
	assert.Equal(t, "stdout", logs[0].Stream)
	assert.Equal(t, "Log message", logs[0].Message)

	input2 := "2024-06-01T12:34:56Z stderr Error message"
	logs2 := parseDockerLogs(input2)
	assert.Len(t, logs2, 1)
	assert.Equal(t, "stderr", logs2[0].Stream)
	assert.Equal(t, "Error message", logs2[0].Message)
}

func TestBuildLogs_Management(t *testing.T) {
	bm := NewBuildManager()
	deploymentID := "dep_test123"

	// Add logs
	bm.addBuildLog(deploymentID, "info", "Starting build")
	bm.addBuildLog(deploymentID, "info", "Cloning repository")
	bm.addBuildLog(deploymentID, "error", "Build failed")

	// Retrieve logs
	logs := bm.GetBuildLogs(deploymentID)
	assert.Len(t, logs, 3)
	assert.Equal(t, "info", logs[0].Level)
	assert.Equal(t, "Starting build", logs[0].Message)
	assert.Equal(t, "error", logs[2].Level)
	assert.Equal(t, "Build failed", logs[2].Message)

	// All logs should have timestamps
	for _, l := range logs {
		assert.NotEmpty(t, l.Timestamp)
	}

	// Cleanup
	bm.CleanupBuildLogs(deploymentID)
	logs = bm.GetBuildLogs(deploymentID)
	assert.Nil(t, logs)
}

func TestBuildLogs_Concurrent(t *testing.T) {
	bm := NewBuildManager()
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			depID := "dep_concurrent_" + string(rune('0'+id))
			for j := 0; j < 10; j++ {
				bm.addBuildLog(depID, "info", "log entry")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// All deployments should have exactly 10 logs each
	for i := 0; i < 10; i++ {
		depID := "dep_concurrent_" + string(rune('0'+i))
		logs := bm.GetBuildLogs(depID)
		assert.Len(t, logs, 10, "Deployment %s has %d logs", depID, len(logs))
	}
}

func TestBuildManager_SetDataDir(t *testing.T) {
	bm := NewBuildManager()
	assert.Equal(t, "/var/panel", bm.dataDir)

	bm.SetDataDir("/custom/data")
	assert.Equal(t, "/custom/data", bm.dataDir)
}

func TestContainerManager_SetDataDir(t *testing.T) {
	cm := NewContainerManager()
	assert.Equal(t, "/var/panel", cm.dataDir)
	assert.Equal(t, "panel_apps", cm.networkName)

	cm.SetDataDir("/custom/data")
	assert.Equal(t, "/custom/data", cm.dataDir)

	cm.SetNetworkName("custom_network")
	assert.Equal(t, "custom_network", cm.networkName)
}

func TestBuildLogs_AddLog(t *testing.T) {
	bm := NewBuildManager()
	result := &BuildResult{
		BuildLogs: make([]LogLine, 0),
	}

	bm.addLog(result, "info", "test message")
	assert.Len(t, result.BuildLogs, 1)
	assert.Equal(t, "info", result.BuildLogs[0].Level)
	assert.Equal(t, "test message", result.BuildLogs[0].Message)

	bm.addLog(result, "error", "error message")
	assert.Len(t, result.BuildLogs, 2)
	assert.Equal(t, "error", result.BuildLogs[1].Level)
}

func TestContainerNameFormat(t *testing.T) {
	appID := "app_abc123def456"
	expectedPrefix := "panel-"
	containerName := expectedPrefix + appID
	assert.True(t, strings.HasPrefix(containerName, "panel-"))
	assert.Equal(t, "panel-app_abc123def456", containerName)
}

func TestDockerLogParsing_EdgeCases(t *testing.T) {
	// Empty line
	logs := parseDockerLogs("")
	assert.Empty(t, logs)

	// Just timestamp, no message
	logs = parseDockerLogs("2024-01-01T00:00:00Z")
	assert.Empty(t, logs)

	// Multiple stdout/stderr
	input := "2024-01-01T00:00:00Z stdout Normal log\n2024-01-01T00:00:01Z stderr Warning log"
	logs = parseDockerLogs(input)
	assert.Len(t, logs, 2)
	assert.Equal(t, "stdout", logs[0].Stream)
	assert.Equal(t, "stderr", logs[1].Stream)
}

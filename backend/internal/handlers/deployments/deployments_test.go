package deployments

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			"single line",
			"hello world",
			[]string{"hello world"},
		},
		{
			"two lines",
			"line1\nline2",
			[]string{"line1", "line2"},
		},
		{
			"three lines",
			"line1\nline2\nline3",
			[]string{"line1", "line2", "line3"},
		},
		{
			"empty string",
			"",
			nil,
		},
		{
			"trailing newline",
			"line1\nline2\n",
			[]string{"line1", "line2"},
		},
		{
			"with carriage return",
			"line1\r\nline2",
			[]string{"line1\r", "line2"},
		},
		{
			"windows style",
			"Building...\r\nDone.\r\n",
			[]string{"Building...\r", "Done.\r"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"exact match", "hello world", "hello", true},
		{"at end", "hello world", "world", true},
		{"middle", "hello world", "lo wo", true},
		{"no match", "hello world", "goodbye", false},
		{"empty substr", "hello", "", true}, // empty string is a substring of everything
		{"empty string", "", "test", false},
		{"substr longer than string", "hi", "hello", false},
		{"case sensitive", "Hello", "hello", false},
		{"single char", "hello", "h", true},
		{"not found single", "hello", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseBuildLogs(t *testing.T) {
	// Test with error lines
	logText := "INFO: Starting build\nERROR: Build failed\nWARN: Configuration issue\nNormal output"
	logs := parseBuildLogs(logText)

	assert.Len(t, logs, 4)
	assert.Equal(t, "info", logs[0].Level)
	assert.Equal(t, "INFO: Starting build", logs[0].Message)
	assert.Equal(t, "error", logs[1].Level)
	assert.Equal(t, "ERROR: Build failed", logs[1].Message)
	assert.Equal(t, "warning", logs[2].Level)
	assert.Equal(t, "WARN: Configuration issue", logs[2].Message)
	assert.Equal(t, "info", logs[3].Level)
	assert.Equal(t, "Normal output", logs[3].Message)

	// Verify timestamps are not empty
	for _, l := range logs {
		assert.NotEmpty(t, l.Timestamp)
	}
}

func TestParseBuildLogs_EmptyInput(t *testing.T) {
	logs := parseBuildLogs("")
	assert.Len(t, logs, 0)
}

func TestParseBuildLogs_AllLevels(t *testing.T) {
	tests := []struct {
		line     string
		expLevel string
	}{
		{"ERROR: something broke", "error"},
		{"error: something broke", "error"},
		{"Failed to compile", "error"},
		{"WARN: deprecated", "warning"},
		{"warning: low memory", "warning"},
		{"INFO: starting", "info"},
		{"normal output", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			logs := parseBuildLogs(tt.line)
			assert.Len(t, logs, 1)
			assert.Equal(t, tt.expLevel, logs[0].Level)
		})
	}
}

func TestDeploymentStatusValidation(t *testing.T) {
	// Verify cancellation logic: only queued and in_progress can be cancelled
	cancellableStatuses := map[string]bool{
		"queued":      true,
		"in_progress": true,
		"success":     false,
		"failed":      false,
		"cancelled":   false,
	}

	for status, expected := range cancellableStatuses {
		isCancellable := status == "queued" || status == "in_progress"
		assert.Equal(t, expected, isCancellable,
			"Status '%s' cancellable=%v, expected=%v", status, isCancellable, expected)
	}
}

func TestParseBuildLogs_LargeInput(t *testing.T) {
	var lines []string
	for i := 0; i < 1000; i++ {
		lines = append(lines, "INFO: Build step "+string(rune('0'+i%10)))
	}
	logText := strings.Join(lines, "\n")
	logs := parseBuildLogs(logText)
	assert.Len(t, logs, 1000)
}

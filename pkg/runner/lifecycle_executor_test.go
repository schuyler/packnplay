package runner

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/obra/packnplay/pkg/devcontainer"
)

// TestLifecycleExecutor_ExecuteString tests executing a string command
func TestLifecycleExecutor_ExecuteString(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// Create a string command
	jsonData := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("onCreate", &cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify docker exec was called with shell command
	if len(mockClient.execCalls) != 1 {
		t.Fatalf("Expected 1 exec call, got %d", len(mockClient.execCalls))
	}

	execArgs := mockClient.execCalls[0]
	// Should be: exec -u testuser test-container sh -c "npm install"
	if !contains(execArgs, "exec") || !contains(execArgs, "test-container") ||
		!contains(execArgs, "sh") || !contains(execArgs, "-c") {
		t.Errorf("Expected docker exec with shell, got: %v", execArgs)
	}
}

// TestLifecycleExecutor_ExecuteArray_ShellCommands tests array where each
// element is a full shell command (first element contains a space).
func TestLifecycleExecutor_ExecuteArray_ShellCommands(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	jsonData := `["npm install", "npm run build"]`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("postCreate", &cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Each element runs as a separate shell command
	if len(mockClient.execCalls) != 2 {
		t.Fatalf("Expected 2 exec calls, got %d", len(mockClient.execCalls))
	}

	for i, call := range mockClient.execCalls {
		if !contains(call, "/bin/sh", "-c") {
			t.Errorf("exec call %d: expected shell execution, got: %v", i, call)
		}
	}
}

// TestLifecycleExecutor_ExecuteArray_DirectExec tests array where elements
// form a single command with arguments (first element has no space).
func TestLifecycleExecutor_ExecuteArray_DirectExec(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	jsonData := `["sh", "-c", "echo hello"]`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("postCreate", &cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Single direct exec call, no shell wrapping
	if len(mockClient.execCalls) != 1 {
		t.Fatalf("Expected 1 exec call, got %d", len(mockClient.execCalls))
	}

	execArgs := mockClient.execCalls[0]
	if !contains(execArgs, "sh", "-c", "echo hello") {
		t.Errorf("Expected direct exec with sh -c, got: %v", execArgs)
	}
}

// TestLifecycleExecutor_ExecuteObject tests executing parallel commands
func TestLifecycleExecutor_ExecuteObject(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// Create an object command with 2 parallel tasks
	jsonData := `{
		"task1": "echo 'task 1'",
		"task2": "echo 'task 2'"
	}`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("postStart", &cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify both commands were executed
	if len(mockClient.execCalls) != 2 {
		t.Fatalf("Expected 2 exec calls for parallel execution, got %d", len(mockClient.execCalls))
	}
}

// TestLifecycleExecutor_ExecuteError tests error handling
func TestLifecycleExecutor_ExecuteError(t *testing.T) {
	mockClient := &mockDockerClient{
		execError: fmt.Errorf("command failed"),
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// Create a command that will fail
	jsonData := `"npm install"`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("onCreate", &cmd)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestLifecycleExecutor_NilCommand tests handling of nil command
func TestLifecycleExecutor_NilCommand(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// Execute nil command (should be no-op)
	err := executor.Execute("onCreate", nil)
	if err != nil {
		t.Fatalf("Expected no error for nil command, got: %v", err)
	}

	// Should not have called exec
	if len(mockClient.execCalls) != 0 {
		t.Errorf("Expected 0 exec calls for nil command, got %d", len(mockClient.execCalls))
	}
}

// TestLifecycleExecutor_ExecuteAllLifecycle tests executing all lifecycle commands
func TestLifecycleExecutor_ExecuteAllLifecycle(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// Create a config with all lifecycle commands
	jsonData := `{
		"image": "ubuntu:22.04",
		"onCreateCommand": "echo 'onCreate'",
		"postCreateCommand": "echo 'postCreate'",
		"postStartCommand": "echo 'postStart'"
	}`

	var config devcontainer.Config
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Execute all lifecycle commands
	if config.OnCreateCommand != nil {
		if err := executor.Execute("onCreate", config.OnCreateCommand); err != nil {
			t.Errorf("onCreate failed: %v", err)
		}
	}

	if config.PostCreateCommand != nil {
		if err := executor.Execute("postCreate", config.PostCreateCommand); err != nil {
			t.Errorf("postCreate failed: %v", err)
		}
	}

	if config.PostStartCommand != nil {
		if err := executor.Execute("postStart", config.PostStartCommand); err != nil {
			t.Errorf("postStart failed: %v", err)
		}
	}

	// Should have executed 3 commands
	if len(mockClient.execCalls) != 3 {
		t.Errorf("Expected 3 exec calls, got %d", len(mockClient.execCalls))
	}
}

// TestLifecycleExecutor_VerboseOutput tests verbose mode
func TestLifecycleExecutor_VerboseOutput(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls:  [][]string{},
		execOutput: "test output",
	}

	// Create executor in verbose mode
	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", true, nil)

	jsonData := `"echo 'test'"`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("postStart", &cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// In verbose mode, output should be captured
	// (actual output testing would require capturing stdout)
}

// Enhanced mockDockerClient with exec tracking
type mockDockerClientWithExec struct {
	mockDockerClient
	execCalls  [][]string
	execOutput string
	execError  error
}

// TestLifecycleExecutor_MultipleParallelErrors tests handling of multiple task failures
func TestLifecycleExecutor_MultipleParallelErrors(t *testing.T) {
	mockClient := &mockDockerClient{
		execError: fmt.Errorf("command failed"),
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// Create an object command with 3 tasks that will all fail
	jsonData := `{
		"task1": "echo 'task 1'",
		"task2": "echo 'task 2'",
		"task3": "echo 'task 3'"
	}`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("postCreate", &cmd)
	if err == nil {
		t.Fatal("Expected error when all tasks fail")
	}

	// Error should mention multiple failures
	errMsg := err.Error()
	if !strings.Contains(errMsg, "multiple tasks failed") {
		t.Errorf("Expected error to mention multiple failures, got: %s", errMsg)
	}

	// Should contain task names
	if !strings.Contains(errMsg, "task") {
		t.Errorf("Expected error to include task names, got: %s", errMsg)
	}
}

// TestLifecycleExecutor_ObjectWithArrayValues tests executing an object command
// where values are arrays of shell commands (as used in packnplay's devcontainer.json).
// Each array element is a full shell command that should be run through /bin/sh -c,
// NOT treated as [executable, arg1, arg2, ...].
func TestLifecycleExecutor_ObjectWithArrayValues(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// This mirrors the format in .devcontainer/devcontainer.json:
	// object with array values where each element is a shell command
	jsonData := `{
		"versions": [
			"echo 'Development Environment Setup Complete'",
			"node --version",
			"python3 --version"
		],
		"ai-tools": [
			"echo 'Installing AI CLI tools...'",
			"npm install -g @anthropic-ai/claude-code",
			"echo 'Done'"
		]
	}`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("onCreate", &cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Each array element should have been executed through a shell.
	// Verify that every exec call includes /bin/sh -c
	for i, call := range mockClient.execCalls {
		callStr := strings.Join(call, " ")
		if !strings.Contains(callStr, "/bin/sh") || !strings.Contains(callStr, "-c") {
			t.Errorf("exec call %d was not run through shell: %v", i, call)
		}
	}

	// The first element of each array should NOT be treated as an executable path.
	// If we see "echo 'Development Environment Setup Complete'" as a direct arg
	// after the container name (without /bin/sh -c before it), that's the bug.
	for i, call := range mockClient.execCalls {
		// Find the container name position, everything after it is the command
		for j, arg := range call {
			if arg == "test-container" && j+1 < len(call) {
				nextArg := call[j+1]
				if nextArg != "/bin/sh" {
					t.Errorf("exec call %d: expected /bin/sh after container name, got %q — command is being executed without a shell", i, nextArg)
				}
				break
			}
		}
	}
}

// TestLifecycleExecutor_TopLevelArrayOfShellCommands tests a top-level array
// where each element is a full shell command (not [command, arg1, arg2, ...]).
func TestLifecycleExecutor_TopLevelArrayOfShellCommands(t *testing.T) {
	mockClient := &mockDockerClient{
		execCalls: [][]string{},
	}

	executor := NewLifecycleExecutor(mockClient, "test-container", "testuser", false, nil)

	// This mirrors postCreateCommand from .devcontainer/devcontainer.json
	jsonData := `[
		"echo 'container ready'",
		"echo 'all tools configured'"
	]`
	var cmd devcontainer.LifecycleCommand
	if err := cmd.UnmarshalJSON([]byte(jsonData)); err != nil {
		t.Fatalf("Failed to unmarshal command: %v", err)
	}

	err := executor.Execute("postCreate", &cmd)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Each element should be run as a separate shell command.
	// The first element should NOT be treated as the executable path.
	for i, call := range mockClient.execCalls {
		for j, arg := range call {
			if arg == "test-container" && j+1 < len(call) {
				nextArg := call[j+1]
				if nextArg != "/bin/sh" {
					t.Errorf("exec call %d: expected /bin/sh after container name, got %q — command is being executed without a shell", i, nextArg)
				}
				break
			}
		}
	}
}

// contains checks if a string slice contains all the given strings
func contains(slice []string, strs ...string) bool {
	sliceStr := strings.Join(slice, " ")
	for _, s := range strs {
		if !strings.Contains(sliceStr, s) {
			return false
		}
	}
	return true
}

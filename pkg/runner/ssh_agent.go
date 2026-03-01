package runner

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// findSSHAgentSocket returns the SSH agent socket path that can be mounted
// into a Docker container. The returned path is resolvable from within the
// Docker VM (or directly on the host for native Linux Docker).
func findSSHAgentSocket() (string, error) {
	if runtime.GOOS == "linux" {
		return findSSHAgentSocketLinux()
	}
	if runtime.GOOS == "darwin" {
		return findSSHAgentSocketDarwin()
	}
	return "", fmt.Errorf("SSH agent forwarding is not supported on %s", runtime.GOOS)
}

// findSSHAgentSocketLinux returns the host's SSH_AUTH_SOCK directly,
// since on Linux the Docker daemon runs natively and can mount host sockets.
func findSSHAgentSocketLinux() (string, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return "", fmt.Errorf("SSH_AUTH_SOCK is not set; is ssh-agent running?")
	}
	return sock, nil
}

// findSSHAgentSocketDarwin detects the Docker runtime on macOS and returns
// the appropriate socket path. On macOS, Docker runs inside a VM, so the
// socket path must be resolvable from within that VM.
func findSSHAgentSocketDarwin() (string, error) {
	dockerHost := os.Getenv("DOCKER_HOST")

	if strings.Contains(dockerHost, "colima") {
		return findColimaSSHSocket()
	}

	// Docker Desktop provides a well-known socket path inside its VM.
	// Requires "Allow the default Docker socket to be used" in Docker Desktop settings.
	// If SSH operations fail, verify SSH agent forwarding is enabled in Docker Desktop.
	return "/run/host-services/ssh-auth.sock", nil
}

// findColimaSSHSocket queries the Colima VM for its SSH_AUTH_SOCK path.
// Requires Colima to be started with --ssh-agent (or forwardAgent: true in config).
func findColimaSSHSocket() (string, error) {
	cmd := exec.Command("colima", "ssh", "--", "printenv", "SSH_AUTH_SOCK")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not get SSH agent socket from Colima VM.\n" +
			"Ensure Colima is running with SSH agent forwarding:\n" +
			"  colima start --ssh-agent")
	}

	sock := strings.TrimSpace(string(output))
	if sock == "" {
		return "", fmt.Errorf("SSH_AUTH_SOCK is not set in the Colima VM.\n" +
			"Restart Colima with SSH agent forwarding:\n" +
			"  colima stop && colima start --ssh-agent")
	}

	return sock, nil
}

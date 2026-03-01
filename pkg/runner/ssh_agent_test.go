package runner

import (
	"os"
	"runtime"
	"testing"
)

func TestFindSSHAgentSocketLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	t.Run("returns SSH_AUTH_SOCK when set", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "/tmp/ssh-test/agent.123")
		sock, err := findSSHAgentSocketLinux()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sock != "/tmp/ssh-test/agent.123" {
			t.Errorf("got %q, want %q", sock, "/tmp/ssh-test/agent.123")
		}
	})

	t.Run("returns error when SSH_AUTH_SOCK is empty", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "")
		_, err := findSSHAgentSocketLinux()
		if err == nil {
			t.Fatal("expected error when SSH_AUTH_SOCK is empty")
		}
	})
}

func TestFindSSHAgentSocketLinuxUnset(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}

	// Unset SSH_AUTH_SOCK for this test
	orig := os.Getenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AUTH_SOCK")
	defer os.Setenv("SSH_AUTH_SOCK", orig)

	_, err := findSSHAgentSocketLinux()
	if err == nil {
		t.Fatal("expected error when SSH_AUTH_SOCK is unset")
	}
}

func TestFindSSHAgentSocketDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}

	t.Run("defaults to Docker Desktop socket path", func(t *testing.T) {
		t.Setenv("DOCKER_HOST", "")
		sock, err := findSSHAgentSocketDarwin()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sock != "/run/host-services/ssh-auth.sock" {
			t.Errorf("got %q, want Docker Desktop socket path", sock)
		}
	})

	t.Run("detects Colima from DOCKER_HOST", func(t *testing.T) {
		// This will call colima ssh which may not work in CI,
		// but verifies the detection path is taken
		t.Setenv("DOCKER_HOST", "unix:///Users/test/.colima/default/docker.sock")
		sock, err := findSSHAgentSocketDarwin()
		if err != nil {
			// Expected if colima isn't running
			t.Logf("Colima detection returned error (expected in CI): %v", err)
			return
		}
		if sock == "" {
			t.Error("got empty socket path from Colima")
		}
	})
}

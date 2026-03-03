package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildBinary builds the test binary and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "git-credential-1password.exe")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, output)
	}
	return binary
}

func TestVersionFlag(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "git-credential-1password") {
		t.Errorf("version output = %q, want to contain 'git-credential-1password'", output)
	}
}

func TestNoArguments(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.ExitCode())
	}
	if !strings.Contains(string(output), "usage:") {
		t.Errorf("output = %q, want to contain 'usage:'", output)
	}
}

func TestUnknownSubcommand(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "unknown")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(string(output), "Unknown argument") {
		t.Errorf("output = %q, want to contain 'Unknown argument'", output)
	}
}

func TestGetVersion(t *testing.T) {
	got := getVersion()
	if got == "" {
		t.Error("getVersion() returned empty string")
	}
}

func TestReadOnlyStoreIsNoop(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "-read-only", "store")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=example.com\n\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, output)
	}
}

func TestReadOnlyEraseIsNoop(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "-read-only", "erase")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=example.com\n\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, output)
	}
}

func TestEraseWithoutFlag(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "erase")
	cmd.Stdin = strings.NewReader("protocol=https\nhost=example.com\n\n")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(string(output), "-erase true") {
		t.Errorf("output = %q, want to contain '-erase true'", output)
	}
}

func TestTooManyArguments(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "get", "extra")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.ExitCode())
	}
}

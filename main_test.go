package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testBinary is built once in TestMain and shared by all tests.
var testBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "git-credential-1password-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testBinary = filepath.Join(dir, "git-credential-1password.exe")
	output, err := exec.Command("go", "build", "-o", testBinary, ".").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary: %v\n%s", err, output)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// buildBinary returns the path of the binary built in TestMain.
func buildBinary(t *testing.T) string {
	t.Helper()
	return testBinary
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
	exitErr, ok := errors.AsType[*exec.ExitError](err)
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
	// git runs erase by itself after a failed login, so this must not fail
	if err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "--erase") {
		t.Errorf("output = %q, want to contain '--erase'", output)
	}
	if strings.Contains(string(output), "usage:") {
		t.Errorf("output = %q, want no usage text", output)
	}
}

func TestErrorsHaveNoTimestamp(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "unknown")
	output, _ := cmd.CombinedOutput()
	if !strings.HasPrefix(string(output), "git-credential-1password: ") {
		t.Errorf("output = %q, want to start with 'git-credential-1password: '", output)
	}
}

func TestTooManyArguments(t *testing.T) {
	binary := buildBinary(t)
	cmd := exec.Command(binary, "get", "extra")
	_, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.ExitCode())
	}
}

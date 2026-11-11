package internal

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// fakeOpEnv selects the fake op behaviour. When set, the test binary acts as
// the op CLI instead of running tests (see TestMain).
const fakeOpEnv = "GIT_CREDENTIAL_1PASSWORD_FAKE_OP"

func TestMain(m *testing.M) {
	switch os.Getenv(fakeOpEnv) {
	case "":
		os.Exit(m.Run())
	case "warn":
		// op prints update notices and warnings to stderr next to valid JSON
		fmt.Fprintln(os.Stderr, "[WARN] a new version of op is available")
		fmt.Println(`[{"id":"abc123","urls":[{"href":"https://example.com"}]}]`)
		os.Exit(0)
	case "fail":
		fmt.Fprintln(os.Stderr, "[ERROR] item not found")
		os.Exit(1)
	}
}

// useFakeOp points OpPath at the test binary running in the given mode.
func useFakeOp(t *testing.T, mode string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	setupTest(t, &mockRunner{})
	OpPath = exe
	t.Setenv(fakeOpEnv, mode)
}

func TestExecOpRunnerIgnoresStderrOnSuccess(t *testing.T) {
	useFakeOp(t, "warn")

	result, err := (&ExecOpRunner{}).ListItems()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item := result.FindByURL("https://example.com"); item == nil || item.Id != "abc123" {
		t.Errorf("FindByURL() = %+v, want id abc123", item)
	}
}

func TestExecOpRunnerIncludesStderrOnFailure(t *testing.T) {
	useFakeOp(t, "fail")
	r := &ExecOpRunner{}

	_, listErr := r.ListItems()
	_, getErr := r.GetItem("abc123")
	_, createErr := r.CreateItem()
	_, editErr := r.EditItem("abc123")
	deleteErr := r.DeleteItem("abc123")

	for name, err := range map[string]error{
		"ListItems":  listErr,
		"GetItem":    getErr,
		"CreateItem": createErr,
		"EditItem":   editErr,
		"DeleteItem": deleteErr,
	} {
		if err == nil {
			t.Errorf("%s: expected error, got nil", name)
			continue
		}
		if !strings.Contains(err.Error(), "[ERROR] item not found") {
			t.Errorf("%s: error = %q, want op stderr", name, err)
		}
	}
}

func TestFindItemIdErrorNotDoublePrefixed(t *testing.T) {
	useFakeOp(t, "fail")
	Runner = &ExecOpRunner{}

	_, err := findItemId("https", "example.com")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if n := strings.Count(err.Error(), "op item list"); n != 1 {
		t.Errorf("error = %q, want exactly one 'op item list' prefix", err)
	}
}

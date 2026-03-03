package internal

import (
	"fmt"
	"runtime"
	"testing"
)

func TestGetField(t *testing.T) {
	tests := []struct {
		name     string
		item     OpItem
		label    string
		expected string
	}{
		{
			name: "field exists",
			item: OpItem{
				{Label: "username", Value: "alice"},
				{Label: "password", Value: "secret"},
			},
			label:    "username",
			expected: "alice",
		},
		{
			name: "field missing",
			item: OpItem{
				{Label: "username", Value: "alice"},
			},
			label:    "password",
			expected: "",
		},
		{
			name:     "empty item",
			item:     OpItem{},
			label:    "username",
			expected: "",
		},
		{
			name: "duplicate labels returns first",
			item: OpItem{
				{Label: "username", Value: "first"},
				{Label: "username", Value: "second"},
			},
			label:    "username",
			expected: "first",
		},
		{
			name: "empty label matches empty",
			item: OpItem{
				{Label: "", Value: "val"},
			},
			label:    "",
			expected: "val",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.item.GetField(tt.label)
			if got != tt.expected {
				t.Errorf("GetField(%q) = %q, want %q", tt.label, got, tt.expected)
			}
		})
	}
}

func TestFindByURL(t *testing.T) {
	tests := []struct {
		name    string
		result  OpItemListResult
		url     string
		wantID  string
		wantNil bool
	}{
		{
			name: "URL match",
			result: OpItemListResult{
				{Id: "abc123", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			url:    "https://github.com",
			wantID: "abc123",
		},
		{
			name: "no match",
			result: OpItemListResult{
				{Id: "abc123", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			url:     "https://gitlab.com",
			wantNil: true,
		},
		{
			name:    "empty list",
			result:  OpItemListResult{},
			url:     "https://github.com",
			wantNil: true,
		},
		{
			name: "multiple URLs per item",
			result: OpItemListResult{
				{Id: "abc123", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{
					{Href: "https://github.com"},
					{Href: "https://api.github.com"},
				}},
			},
			url:    "https://api.github.com",
			wantID: "abc123",
		},
		{
			name: "multiple items first match wins",
			result: OpItemListResult{
				{Id: "first", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
				{Id: "second", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			url:    "https://github.com",
			wantID: "first",
		},
		{
			name: "item with no URLs",
			result: OpItemListResult{
				{Id: "abc123", URLs: nil},
			},
			url:     "https://github.com",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.FindByURL(tt.url)
			if tt.wantNil {
				if got != nil {
					t.Errorf("FindByURL(%q) = %+v, want nil", tt.url, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("FindByURL(%q) = nil, want id=%q", tt.url, tt.wantID)
			}
			if got.Id != tt.wantID {
				t.Errorf("FindByURL(%q).Id = %q, want %q", tt.url, got.Id, tt.wantID)
			}
		})
	}
}

func TestItemName(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		host     string
		expected string
	}{
		{"no prefix", "", "github.com", "github.com"},
		{"with prefix", "git-", "github.com", "git-github.com"},
		{"empty host", "prefix-", "", "prefix-"},
		{"both empty", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origPrefix := Prefix
			Prefix = tt.prefix
			defer func() { Prefix = origPrefix }()

			got := itemName(tt.host)
			if got != tt.expected {
				t.Errorf("itemName(%q) = %q, want %q", tt.host, got, tt.expected)
			}
		})
	}
}

func TestOpCommand(t *testing.T) {
	tests := []struct {
		name     string
		opPath   string
		expected string
	}{
		{"custom path", "/usr/local/bin/op", "/usr/local/bin/op"},
	}

	// Add OS-specific test
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name     string
			opPath   string
			expected string
		}{"default on windows", "", "op.exe"})
	} else {
		tests = append(tests, struct {
			name     string
			opPath   string
			expected string
		}{"default on non-windows", "", "op"})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origOpPath := OpPath
			OpPath = tt.opPath
			defer func() { OpPath = origOpPath }()

			got := opCommand()
			if got != tt.expected {
				t.Errorf("opCommand() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildOpItemCommand(t *testing.T) {
	origOpPath := OpPath
	origFlags := OpItemFlags
	defer func() {
		OpPath = origOpPath
		OpItemFlags = origFlags
	}()

	OpPath = "/usr/bin/op"
	OpItemFlags = nil

	t.Run("simple subcommand", func(t *testing.T) {
		cmd := buildOpItemCommand("list", "--format", "json")
		expected := []string{"/usr/bin/op", "item", "list", "--format", "json"}
		if len(cmd.Args) != len(expected) {
			t.Fatalf("Args length = %d, want %d: %v", len(cmd.Args), len(expected), cmd.Args)
		}
		for i, arg := range expected {
			if cmd.Args[i] != arg {
				t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], arg)
			}
		}
	})

	t.Run("with OpItemFlags", func(t *testing.T) {
		OpItemFlags = []string{"--account", "myaccount", "--vault", "myvault"}
		cmd := buildOpItemCommand("get", "item123")
		expected := []string{"/usr/bin/op", "item", "get", "--account", "myaccount", "--vault", "myvault", "item123"}
		if len(cmd.Args) != len(expected) {
			t.Fatalf("Args length = %d, want %d: %v", len(cmd.Args), len(expected), cmd.Args)
		}
		for i, arg := range expected {
			if cmd.Args[i] != arg {
				t.Errorf("Args[%d] = %q, want %q", i, cmd.Args[i], arg)
			}
		}
	})

	t.Run("no extra args", func(t *testing.T) {
		OpItemFlags = nil
		cmd := buildOpItemCommand("list")
		expected := []string{"/usr/bin/op", "item", "list"}
		if len(cmd.Args) != len(expected) {
			t.Fatalf("Args length = %d, want %d: %v", len(cmd.Args), len(expected), cmd.Args)
		}
	})
}

func TestFindItemId(t *testing.T) {
	t.Run("with explicit ItemID", func(t *testing.T) {
		m := &mockRunner{}
		setupTest(t, m)
		ItemID = "explicit-id"

		id, err := findItemId("https", "github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == nil || *id != "explicit-id" {
			t.Errorf("findItemId() = %v, want 'explicit-id'", id)
		}
	})

	t.Run("found via list", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "found-id", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
		}
		setupTest(t, m)

		id, err := findItemId("https", "github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == nil || *id != "found-id" {
			t.Errorf("findItemId() = %v, want 'found-id'", id)
		}
	})

	t.Run("not found", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "other-id", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://gitlab.com"}}},
			},
		}
		setupTest(t, m)

		id, err := findItemId("https", "github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != nil {
			t.Errorf("findItemId() = %v, want nil", *id)
		}
	})

	t.Run("list error", func(t *testing.T) {
		m := &mockRunner{
			listItemsErr: fmt.Errorf("connection refused"),
		}
		setupTest(t, m)

		_, err := findItemId("https", "github.com")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("empty list", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{},
		}
		setupTest(t, m)

		id, err := findItemId("https", "github.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id != nil {
			t.Errorf("findItemId() = %v, want nil", *id)
		}
	})
}

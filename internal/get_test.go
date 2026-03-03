package internal

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestGetCommand(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemResult: OpItem{
				{Label: "username", Value: "alice"},
				{Label: "password", Value: "secret"},
			},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "username=alice\npassword=secret\n\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})

	t.Run("with explicit ItemID", func(t *testing.T) {
		m := &mockRunner{
			getItemResult: OpItem{
				{Label: "username", Value: "bob"},
				{Label: "password", Value: "pass123"},
			},
		}
		setupTest(t, m)
		ItemID = "explicit-id"

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.getItemCalledWith) != 1 || m.getItemCalledWith[0] != "explicit-id" {
			t.Errorf("GetItem called with %v, want [explicit-id]", m.getItemCalledWith)
		}
	})

	t.Run("missing host", func(t *testing.T) {
		m := &mockRunner{}
		setupTest(t, m)

		input := "protocol=https\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "host is missing") {
			t.Errorf("error = %q, want to contain 'host is missing'", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("error = %v, want ErrNotFound", err)
		}
	})

	t.Run("opGetItem error", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemErr: fmt.Errorf("connection timeout"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "connection timeout") {
			t.Errorf("error = %q, want to contain 'connection timeout'", err)
		}
	})

	t.Run("empty username", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemResult: OpItem{
				{Label: "username", Value: ""},
				{Label: "password", Value: "secret"},
			},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "username or password is empty") {
			t.Errorf("error = %q, want to contain 'username or password is empty'", err)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemResult: OpItem{
				{Label: "username", Value: "alice"},
				{Label: "password", Value: ""},
			},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "username or password is empty") {
			t.Errorf("error = %q, want to contain 'username or password is empty'", err)
		}
	})

	t.Run("list items error", func(t *testing.T) {
		m := &mockRunner{
			listItemsErr: fmt.Errorf("op not found"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		m := &mockRunner{}
		setupTest(t, m)

		input := "malformed-line\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "reading input") {
			t.Errorf("error = %q, want to contain 'reading input'", err)
		}
	})

	t.Run("custom field names", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemResult: OpItem{
				{Label: "email", Value: "alice@example.com"},
				{Label: "token", Value: "ghp_abc123"},
			},
		}
		setupTest(t, m)
		UsernameField = "email"
		PasswordField = "token"

		input := "protocol=https\nhost=github.com\n\n"
		var buf bytes.Buffer
		err := GetCommand(strings.NewReader(input), &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "username=alice@example.com\npassword=ghp_abc123\n\n"
		if buf.String() != expected {
			t.Errorf("output = %q, want %q", buf.String(), expected)
		}
	})
}

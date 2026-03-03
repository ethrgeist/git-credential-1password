package internal

import (
	"fmt"
	"strings"
	"testing"
)

func TestEraseCommand(t *testing.T) {
	t.Run("item found and deleted", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		err := EraseCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.deleteItemCalledWith) != 1 {
			t.Fatalf("DeleteItem called %d times, want 1", len(m.deleteItemCalledWith))
		}
		if m.deleteItemCalledWith[0] != "item-1" {
			t.Errorf("DeleteItem called with %q, want 'item-1'", m.deleteItemCalledWith[0])
		}
	})

	t.Run("item not found", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		err := EraseCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.deleteItemCalledWith) != 0 {
			t.Errorf("DeleteItem called %d times, want 0", len(m.deleteItemCalledWith))
		}
	})

	t.Run("delete error", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			deleteItemErr: fmt.Errorf("permission denied"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		err := EraseCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "op item delete failed") {
			t.Errorf("error = %q, want to contain 'op item delete failed'", err)
		}
	})

	t.Run("list items error", func(t *testing.T) {
		m := &mockRunner{
			listItemsErr: fmt.Errorf("op not found"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\n\n"
		err := EraseCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		m := &mockRunner{}
		setupTest(t, m)

		input := "malformed\n"
		err := EraseCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("with explicit ItemID", func(t *testing.T) {
		m := &mockRunner{}
		setupTest(t, m)
		ItemID = "explicit-id"

		input := "protocol=https\nhost=github.com\n\n"
		err := EraseCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.deleteItemCalledWith) != 1 {
			t.Fatalf("DeleteItem called %d times, want 1", len(m.deleteItemCalledWith))
		}
		if m.deleteItemCalledWith[0] != "explicit-id" {
			t.Errorf("DeleteItem called with %q, want 'explicit-id'", m.deleteItemCalledWith[0])
		}
	})
}

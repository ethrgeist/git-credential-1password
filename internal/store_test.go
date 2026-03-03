package internal

import (
	"fmt"
	"strings"
	"testing"
)

func TestStoreCommand(t *testing.T) {
	t.Run("create new item", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.createItemCalledWith) != 1 {
			t.Fatalf("CreateItem called %d times, want 1", len(m.createItemCalledWith))
		}
		args := m.createItemCalledWith[0]
		// Verify key arguments are present
		found := map[string]bool{}
		for _, arg := range args {
			if strings.HasPrefix(arg, "--category=") {
				found["category"] = true
			}
			if strings.HasPrefix(arg, "--tags=") {
				found["tags"] = true
				if arg != "--tags="+TagMarker {
					t.Errorf("tags arg = %q, want --tags=%s", arg, TagMarker)
				}
			}
			if strings.HasPrefix(arg, "--title=") {
				found["title"] = true
			}
			if strings.HasPrefix(arg, "--url=") {
				found["url"] = true
				if arg != "--url=https://github.com" {
					t.Errorf("url arg = %q, want --url=https://github.com", arg)
				}
			}
			if arg == "username=alice" {
				found["username"] = true
			}
			if arg == "password=secret" {
				found["password"] = true
			}
		}
		for _, key := range []string{"category", "tags", "title", "url", "username", "password"} {
			if !found[key] {
				t.Errorf("missing %s in CreateItem args: %v", key, args)
			}
		}
	})

	t.Run("create error", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{},
			createItemErr:   fmt.Errorf("access denied"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "op item create failed") {
			t.Errorf("error = %q, want to contain 'op item create failed'", err)
		}
	})

	t.Run("no-op when credentials unchanged", func(t *testing.T) {
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

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.editItemCalledWith) != 0 {
			t.Errorf("EditItem called %d times, want 0", len(m.editItemCalledWith))
		}
	})

	t.Run("edit when password changed", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemResult: OpItem{
				{Label: "username", Value: "alice"},
				{Label: "password", Value: "old-secret"},
			},
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=new-secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.editItemCalledWith) != 1 {
			t.Fatalf("EditItem called %d times, want 1", len(m.editItemCalledWith))
		}
		args := m.editItemCalledWith[0]
		// First arg should be the item ID
		if args[0] != "item-1" {
			t.Errorf("EditItem first arg = %q, want 'item-1'", args[0])
		}
	})

	t.Run("edit when username changed", func(t *testing.T) {
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

		input := "protocol=https\nhost=github.com\nusername=bob\npassword=secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(m.editItemCalledWith) != 1 {
			t.Fatalf("EditItem called %d times, want 1", len(m.editItemCalledWith))
		}
	})

	t.Run("edit error", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemResult: OpItem{
				{Label: "username", Value: "alice"},
				{Label: "password", Value: "old"},
			},
			editItemErr: fmt.Errorf("network error"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=new\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "op item edit failed") {
			t.Errorf("error = %q, want to contain 'op item edit failed'", err)
		}
	})

	t.Run("get item error during update", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{
				{Id: "item-1", URLs: []struct {
					Href string `json:"href,omitempty"`
				}{{Href: "https://github.com"}}},
			},
			getItemErr: fmt.Errorf("item corrupted"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "op item get failed") {
			t.Errorf("error = %q, want to contain 'op item get failed'", err)
		}
	})

	t.Run("list items error", func(t *testing.T) {
		m := &mockRunner{
			listItemsErr: fmt.Errorf("op not found"),
		}
		setupTest(t, m)

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		m := &mockRunner{}
		setupTest(t, m)

		input := "malformed\n"
		err := StoreCommand(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("with prefix in title", func(t *testing.T) {
		m := &mockRunner{
			listItemsResult: &OpItemListResult{},
		}
		setupTest(t, m)
		Prefix = "git-"

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		args := m.createItemCalledWith[0]
		var titleFound bool
		for _, arg := range args {
			if arg == "--title=git-github.com" {
				titleFound = true
			}
		}
		if !titleFound {
			t.Errorf("expected title 'git-github.com' in args: %v", args)
		}
	})

	t.Run("with explicit ItemID skips create", func(t *testing.T) {
		m := &mockRunner{
			getItemResult: OpItem{
				{Label: "username", Value: "alice"},
				{Label: "password", Value: "old"},
			},
		}
		setupTest(t, m)
		ItemID = "explicit-id"

		input := "protocol=https\nhost=github.com\nusername=alice\npassword=new\n\n"
		err := StoreCommand(strings.NewReader(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should edit, not create
		if len(m.createItemCalledWith) != 0 {
			t.Errorf("CreateItem called %d times, want 0", len(m.createItemCalledWith))
		}
		if len(m.editItemCalledWith) != 1 {
			t.Fatalf("EditItem called %d times, want 1", len(m.editItemCalledWith))
		}
	})
}

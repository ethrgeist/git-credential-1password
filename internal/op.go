package internal

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
)

var (
	Account       string
	Vault         string
	Category      string
	Prefix        string
	UsernameField string
	PasswordField string
	AllowErase    bool
	ReadOnly      bool
	OpPath        string
	ItemID        string
	OpItemFlags   []string
)

const (
	TagMarker = "git-credential-1password"
)

// OpRunner is the interface for interacting with the 1Password CLI.
type OpRunner interface {
	ListItems() (*OpItemListResult, error)
	GetItem(id string) (OpItem, error)
	CreateItem(args ...string) ([]byte, error)
	EditItem(args ...string) ([]byte, error)
	DeleteItem(id string) error
}

// Runner is the package-level OpRunner used by command functions.
// It defaults to ExecOpRunner which calls the real op CLI.
var Runner OpRunner = &ExecOpRunner{}

// ExecOpRunner implements OpRunner by executing the op CLI.
type ExecOpRunner struct{}

// OpItemField is a field in the output of "op item get --format json" command
// we are only interested in key value pairs from fields as label and value
// to get the username and password, nothing else
// Reference: https://support.1password.com/command-line-reference/#item-get
type OpItemField struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
}

// OpItem is the result of an `op item get` command
type OpItem []OpItemField

// OpItemListResultItem is a single item in the result of an `op item list` command (only id is interesting)
type OpItemListResultItem struct {
	Id   string `json:"id,omitempty"`
	URLs []struct {
		Href string `json:"href,omitempty"`
	} `json:"urls,omitempty"`
}

// OpItemListResult is the result of an `op item list` command
type OpItemListResult []OpItemListResultItem

// FindByURL is the search result for a specific host
func (r OpItemListResult) FindByURL(url string) *OpItemListResultItem {
	for item := range r {
		for _, urlItem := range r[item].URLs {
			if urlItem.Href == url {
				return &r[item]
			}
		}
	}
	return nil
}

// GetField returns the value of the field with the given label
func (i OpItem) GetField(label string) string {
	for _, field := range i {
		if field.Label == label {
			return field.Value
		}
	}
	return ""
}

// itemName gets 1password item name
func itemName(host string) string {
	return fmt.Sprintf("%s%s", Prefix, host)
}

// opCommand returns the path to the op binary
func opCommand() string {
	if OpPath == "" && runtime.GOOS == "windows" {
		return "op.exe" // Default to using "op" from PATH, but with .exe suffix on Windows
	} else if OpPath == "" {
		return "op" // Default to using "op" from PATH
	}
	return OpPath
}

// buildOpItemCommand builds an exec.Cmd for "op item" sub command including additional flags
func buildOpItemCommand(subcommand string, args ...string) *exec.Cmd {
	cmdArgs := []string{"item", subcommand}
	cmdArgs = append(cmdArgs, OpItemFlags...)
	cmdArgs = append(cmdArgs, args...)
	return exec.Command(opCommand(), cmdArgs...)
}

// ListItems runs "op item list --format json" command to get all items with their ids.
func (e *ExecOpRunner) ListItems() (*OpItemListResult, error) {
	cmd := buildOpItemCommand("list", "--categories", Category, "--format", "json", "--tags", TagMarker)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("op item list failed: %s\n%s", err, output)
	}

	var result OpItemListResult
	if err = json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("json.Unmarshal() failed: %s", err)
	}

	return &result, nil
}

// GetItem runs "op item get --format json" command with the given id.
func (e *ExecOpRunner) GetItem(id string) (OpItem, error) {
	fields := fmt.Sprintf("%s,%s", UsernameField, PasswordField)
	cmd := buildOpItemCommand("get", "--format", "json", "--reveal", "--fields", fields, id)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("op item get failed: %s\n%s", err, output)
	}

	var opItem OpItem
	if err = json.Unmarshal(output, &opItem); err != nil {
		return nil, fmt.Errorf("json.Unmarshal() failed: %s", err)
	}
	return opItem, nil
}

// CreateItem runs "op item create" command.
func (e *ExecOpRunner) CreateItem(args ...string) ([]byte, error) {
	cmd := buildOpItemCommand("create", args...)
	return cmd.CombinedOutput()
}

// EditItem runs "op item edit" command.
func (e *ExecOpRunner) EditItem(args ...string) ([]byte, error) {
	cmd := buildOpItemCommand("edit", args...)
	return cmd.CombinedOutput()
}

// DeleteItem runs "op item delete" command.
func (e *ExecOpRunner) DeleteItem(id string) error {
	return buildOpItemCommand("delete", id).Run()
}

// findItemId finds the item id for a given host.
func findItemId(protocol string, host string) (*string, error) {
	// If an explicit item ID was provided via --id, use it directly
	// and skip the list+filter lookup entirely.
	if ItemID != "" {
		return &ItemID, nil
	}

	itemList, err := Runner.ListItems()
	if err != nil {
		return nil, fmt.Errorf("op item list failed: %w", err)
	}

	item := itemList.FindByURL(protocol + "://" + host)
	if item == nil {
		return nil, nil
	}

	return &item.Id, nil
}

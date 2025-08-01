package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

var (
	Account       string
	Vault         string
	Prefix        string
	UsernameField string
	PasswordField string
	AllowErase    bool
	OpPath        string
	OpItemFlags   []string
)

const (
	TAG_MARKER = "git-credential-1password"
)

// OpItemField is a field in the output of "op item get --format json" command
// we are only interessted in key value pairs from fields as label and value
// to get the username and password, nothing else
// Reference: https://support.1password.com/command-line-reference/#item-get
type OpItemField struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
}

// result of an `op item get` command
type OpItem []OpItemField

// single item in the result of an `op item list` command (only id is interessting)
type OpItemListResultItem struct {
	Id   string `json:"id,omitempty"`
	URLs []struct {
		Href string `json:"href,omitempty"`
	} `json:"urls,omitempty"`
}

// result of an `op item list` command
type OpItemListResult []OpItemListResultItem

// search result for a specific host
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

// get 1password item name
func itemName(host string) string {
	return fmt.Sprintf("%s%s", Prefix, host)
}

// opCommand returns the path to the op binary
func opCommand() string {
	if OpPath == "" {
		return "op" // Default to using "op" from PATH
	}
	return OpPath
}

// build a exec.Cmd for "op item" sub command including additional flags
func buildOpItemCommand(subcommand string, args ...string) *exec.Cmd {
	cmdArgs := []string{"item", subcommand}
	cmdArgs = append(cmdArgs, OpItemFlags...)
	cmdArgs = append(cmdArgs, args...)
	return exec.Command(opCommand(), cmdArgs...)
}

// opListItems runs "op list items --format json" command to get all items with their ids
func opListItems() (*OpItemListResult, error) {
	cmd := buildOpItemCommand("list", "--categories", "login", "--format", "json", "--tags", TAG_MARKER)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("opListItems failed with %s\n%+s", err, output)
	}

	var result OpItemListResult
	if err = json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("json.Unmarshal() failed with %s", err)
	}

	return &result, nil
}

// opGetItem runs "op item get --format json" command with the given name
func opGetItem(n string) (OpItem, error) {
	// --fields username,password limits the output to only username and password
	fields := fmt.Sprintf("%s,%s", UsernameField, PasswordField)
	opItemGet := buildOpItemCommand("get", "--format", "json", "--reveal", "--fields", fields, n)
	opItemRaw, err := opItemGet.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("opItemGet failed with %s\n%+s", err, opItemRaw)
	}

	// marhsal the raw output to OpItem struct
	var opItem OpItem
	if err = json.Unmarshal(opItemRaw, &opItem); err != nil {
		return nil, fmt.Errorf("json.Unmarshal() failed with %s", err)
	}
	return opItem, nil
}

// find the item id for a given host
func findItemId(protocol string, host string) *string {
	itemList, err := opListItems()
	if err != nil {
		log.Fatalf("op item list failed with %s", err)
	}

	item := itemList.FindByURL(protocol + "://" + host)
	if item == nil {
		return nil
	}

	return &item.Id
}

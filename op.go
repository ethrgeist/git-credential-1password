package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

var (
	tags        string
	prefix      string
	opItemFlags []string
)

// OpItem is the struct for the output of "op item get --format json" command
// we are only interessted in key value pairs from fields as label and value
// to get the username and password, nothing else
// Reference: https://support.1password.com/command-line-reference/#item-get
type OpItem struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
}

type OpItemList []OpItem

// GetField returns the value of the field with the given label
func (i OpItemList) GetField(label string) string {
	for _, field := range i {
		if field.Label == label {
			return field.Value
		}
	}
	return ""
}

// get 1password item name
func itemName(host string) string {
	return fmt.Sprintf("%s%s", prefix, host)
}

// build a exec.Cmd for "op item" sub command including additional flags
func buildOpItemCommand(subcommand string, args ...string) *exec.Cmd {
	cmdArgs := []string{"item", subcommand}
	cmdArgs = append(cmdArgs, opItemFlags...)
	cmdArgs = append(cmdArgs, args...)
	return exec.Command("op", cmdArgs...)
}

// opGetItem runs "op item get --format json" command with the given name
func opGetItem(n string) (OpItemList, error) {
	// --fields username,password limits the output to only username and password
	opItemGet := buildOpItemCommand("get", "--format", "json", "--fields", "username,password", n)
	opItemRaw, err := opItemGet.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("opItemGet failed with %s\n%+s", err, opItemRaw)
	}

	// marhsal the raw output to OpItem struct
	var opItem OpItemList
	if err = json.Unmarshal(opItemRaw, &opItem); err != nil {
		return nil, fmt.Errorf("json.Unmarshal() failed with %s", err)
	}
	return opItem, nil
}

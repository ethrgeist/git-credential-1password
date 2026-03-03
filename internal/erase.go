package internal

import (
	"fmt"
	"io"
)

// EraseCommand deletes a 1Password item based on the provided git input parameters.
func EraseCommand(r io.Reader) error {
	gitInputs, err := ReadLines(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	itemId, err := findItemId(gitInputs["protocol"], gitInputs["host"])
	if err != nil {
		return err
	}
	if itemId != nil {
		// run "op delete item" command with the found item id
		if err := Runner.DeleteItem(*itemId); err != nil {
			return fmt.Errorf("op item delete failed: %w", err)
		}
	}
	return nil
}

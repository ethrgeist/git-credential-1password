package internal

import (
	"fmt"
	"io"
)

// StoreCommand stores or updates credentials in 1Password.
func StoreCommand(r io.Reader) error {
	gitInputs, err := ReadLines(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	itemId, err := findItemId(gitInputs["protocol"], gitInputs["host"])
	if err != nil {
		return err
	}
	if itemId == nil {
		// run "op item create" command with the host value
		userStr := fmt.Sprintf("%s=%s", UsernameField, gitInputs["username"])
		pwStr := fmt.Sprintf("%s=%s", PasswordField, gitInputs["password"])
		_, err := Runner.CreateItem(
			"--category="+Category,
			"--tags="+TagMarker,
			"--title="+itemName(gitInputs["host"]),
			"--url="+gitInputs["protocol"]+"://"+gitInputs["host"],
			userStr, pwStr,
		)
		if err != nil {
			return fmt.Errorf("op item create failed: %w", err)
		}
		return nil
	}

	item, err := Runner.GetItem(*itemId)
	if err != nil {
		return fmt.Errorf("op item get failed: %w", err)
	}

	// only update the item if the username or password has changed
	if item.GetField(UsernameField) != gitInputs["username"] ||
		item.GetField(PasswordField) != gitInputs["password"] {
		// run "op item edit" command to update the item
		// notes:
		//   we don't set --tags here as our marker must be present already if the item was found and there might be other tags present
		//   we don't set --title here as the user might have renamed the item
		//   we don't set --url here as we use it to find the item, and therefore it must be correct already
		userStr := fmt.Sprintf("%s=%s", UsernameField, gitInputs["username"])
		pwStr := fmt.Sprintf("%s=%s", PasswordField, gitInputs["password"])
		_, err := Runner.EditItem(*itemId, userStr, pwStr)
		if err != nil {
			return fmt.Errorf("op item edit failed: %w", err)
		}
	}
	return nil
}

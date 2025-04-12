package internal

import (
	"fmt"
	"log"
)

func StoreCommand() {
	gitInputs := ReadLines()

	itemId := findItemId(gitInputs["protocol"], gitInputs["host"])
	if itemId == nil {
		// run "op item create" command with the host value
		userStr := fmt.Sprintf("%s=%s", UsernameField, gitInputs["username"])
		pwStr := fmt.Sprintf("%s=%s", PasswordField, gitInputs["password"])
		cmd := buildOpItemCommand("create", "--category=Login", "--tags="+TAG_MARKER, "--title="+itemName(gitInputs["host"]), "--url="+gitInputs["protocol"]+"://"+gitInputs["host"], userStr, pwStr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item create failed with %s %s", err, output)
		}
		return
	}

	item, err := opGetItem(*itemId)
	if err != nil {
		log.Fatalf("op item get failed with %s", err)
	}

	// only update the item if the username or password has changed
	if item.GetField(UsernameField) != gitInputs["username"] ||
		item.GetField(PasswordField) != gitInputs["password"] {
		// run "op item edit" command to update the item
		// notes:
		//   we don't set --tags here as our marker must be present already if the item was found and there might be other tags present
		//   we don't set --title here as the user might have renamed the item
		//   we don't set --url here as we use it to find the item and theirfore it must be correct already
		userStr := fmt.Sprintf("%s=%s", UsernameField, gitInputs["username"])
		pwStr := fmt.Sprintf("%s=%s", PasswordField, gitInputs["password"])
		cmd := buildOpItemCommand("edit", *itemId, userStr, pwStr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item edit failed with %s %s", err, output)
		}
	}
}

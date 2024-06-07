package main

import (
	"log"
)

func storeCommand() {
	gitInputs := ReadLines()

	itemId := findItemId(gitInputs["protocol"], gitInputs["host"])
	if itemId == nil {
		// run "op item create" command with the host value
		cmd := buildOpItemCommand("create", "--category=Login", "--tags="+TAG_MARKER, "--title="+itemName(gitInputs["host"]), "--url="+gitInputs["protocol"]+"://"+gitInputs["host"], "username="+gitInputs["username"], "password="+gitInputs["password"])
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item create failed with %s %s", err, output)
		}
	} else {
		// run "op item edit" command to update the item
		// notes:
		//   we don't set --tags here as our marker must be present already if the item was found and there might be other tags present
		//   we don't set --title here as the user might have renamed the item
		//   we don't set --url here as we use it to find the item and theirfore it must be correct already
		cmd := buildOpItemCommand("edit", *itemId, "username="+gitInputs["username"], "password="+gitInputs["password"])
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item edit failed with %s %s", err, output)
		}
	}
}

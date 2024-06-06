package main

import (
	"log"
)

func storeCommand() {
	gitInputs := ReadLines()

	itemId := findItemId(gitInputs["host"])
	if itemId == nil {
		// run "op item create" command with the host value
		cmd := buildOpItemCommand("create", "--category=Login", "--tags="+TAG_MARKER, "--title="+itemName(gitInputs["host"]), "--url="+gitInputs["protocol"]+"://"+gitInputs["host"], "username="+gitInputs["username"], "password="+gitInputs["password"])
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item create failed with %s %s", err, output)
		}
	} else {
		// run "op item edit" command to update the item
		// note we don't set --tags here as our marker must be present already if the item was found and there might be other tags present
		cmd := buildOpItemCommand("edit", *itemId, "--title="+itemName(gitInputs["host"]), "--url="+gitInputs["protocol"]+"://"+gitInputs["host"], "username="+gitInputs["username"], "password="+gitInputs["password"])
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item edit failed with %s %s", err, output)
		}
	}
}

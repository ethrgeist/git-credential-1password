package main

import "log"

func storeCommand() {
	gitInputs := ReadLines()

	item, _ := opGetItem(itemName(gitInputs["host"]))
	if item == nil {
		// run "op create item" command with the host value
		cmd := buildOpItemCommand("create", "--category=Login", "--title="+itemName(gitInputs["host"]), "--url="+gitInputs["protocol"]+"://"+gitInputs["host"], "username="+gitInputs["username"], "password="+gitInputs["password"])
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item create failed with %s %s", err, output)
		}
	} else {
		// run "op create edit" command to update the item
		cmd := buildOpItemCommand("edit", itemName(gitInputs["host"]), "--url="+gitInputs["protocol"]+"://"+gitInputs["host"], "username="+gitInputs["username"], "password="+gitInputs["password"])
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalf("op item edit failed with %s %s", err, output)
		}
	}
}

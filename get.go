package main

import (
	"fmt"
	"log"
	"os"
)

func getCommand() {
	// git sends the input to stdin
	gitInputs := ReadLines()

	// check if the host field is present in the input
	if _, ok := gitInputs["host"]; !ok {
		log.Fatalf("host is missing in the input")
	}

	itemList, err := opListItems()
	if err != nil {
		log.Fatalf("op item list failed with %s", err)
	}

	item := itemList.FindByTitle(itemName(gitInputs["host"]))
	// if the host is not found, we exit with status code 1 to indicate that the host is not found
	// we don't want to print anything to stdout in this case as it is not a real error
	if item == nil {
		os.Exit(1)
	}

	// fetch the item from 1password using the id
	opItem, err := opGetItem(item.Id)
	if err != nil {
		// we bail out if we can't get the item we just listed above - something is wrong and should be reported
		log.Fatalf("op item get failed with %s", err)
	}

	// feed the username and password to git
	username := opItem.GetField("username")
	password := opItem.GetField("password")
	if username == "" || password == "" {
		log.Fatalf("username or password is empty, is the item named correctly?")
	}
	fmt.Printf("username=%s\n", username)
	fmt.Printf("password=%s\n", password)
}

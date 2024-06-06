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

	// run "op item get --format json" command with the host value
	// this can only get, no other operations are allowed
	opItem, err := opGetItem(itemName(gitInputs["host"]))
	if err != nil {
		// if the item is not found, we should exit with 1 to let git know
		// its not an real error, we just don't have the credentials
		os.Exit(1)
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

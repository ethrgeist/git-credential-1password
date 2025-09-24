package internal

import (
	"fmt"
	"log"
	"os"
)

// GetCommand retrieves and prints the username and password from a 1Password item based on git input parameters.
func GetCommand() {
	// git sends the input to stdin
	gitInputs := ReadLines()

	// check if the host field is present in the input
	if _, ok := gitInputs["host"]; !ok {
		log.Fatalf("host is missing in the input")
	}

	itemId := findItemId(gitInputs["protocol"], gitInputs["host"])
	// if the host is not found, we exit with status code 1 to indicate that the host is not found
	// we don't want to print anything to stdout in this case as it is not a real error
	if itemId == nil {
		os.Exit(1)
	}

	// fetch the item from 1password using the id
	opItem, err := opGetItem(*itemId)
	if err != nil {
		// we bail out if we can't get the item we just listed above - something is wrong and should be reported
		log.Fatalf("op item get failed with %s", err)
	}

	// feed the username and password to git
	username := opItem.GetField(UsernameField)
	password := opItem.GetField(PasswordField)
	if username == "" || password == "" {
		log.Fatalf("username or password is empty, is the item named correctly?")
	}
	fmt.Printf("username=%s\npassword=%s\n\n", username, password)
}

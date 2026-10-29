package internal

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrNotFound indicates that the credential was not found in 1Password.
var ErrNotFound = errors.New("credential not found")

// GetCommand retrieves and prints the username and password from a 1Password item
// based on git input parameters. It reads input from r and writes credentials to w.
func GetCommand(r io.Reader, w io.Writer) error {
	// git sends the input to stdin
	gitInputs, err := ReadLines(r)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	// check if the host field is present in the input
	if _, ok := gitInputs["host"]; !ok {
		return fmt.Errorf("host is missing in the input")
	}

	itemId, err := findItemId(gitInputs["protocol"], gitInputs["host"])
	if err != nil {
		return err
	}
	// if the host is not found, we return ErrNotFound
	if itemId == nil {
		return ErrNotFound
	}

	// fetch the item from 1password using the id
	opItem, err := Runner.GetItem(*itemId)
	if err != nil {
		return fmt.Errorf("op item get failed: %w", err)
	}

	// feed the username and password to git
	username := opItem.GetField(UsernameField)
	password := opItem.GetField(PasswordField)
	if username == "" || password == "" {
		return fmt.Errorf("username or password is empty, is the item named correctly?")
	}
	// a newline or NUL would end the value early and inject extra keys into git's input
	if strings.ContainsAny(username, "\n\x00") || strings.ContainsAny(password, "\n\x00") {
		return fmt.Errorf("username or password contains a newline or NUL character")
	}
	fmt.Fprintf(w, "username=%s\npassword=%s\n\n", username, password)
	return nil
}

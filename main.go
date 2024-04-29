package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
)

// OpItem is the struct for the output of "op item get --format json" command
// we are only interessted in key value pairs from fields as label and value
// to get the username and password, nothing else
// Reference: https://support.1password.com/command-line-reference/#item-get
type OpItem struct {
	Label string `json:"label,omitempty"`
	Value string `json:"value,omitempty"`
}

type OpItemList []OpItem

// versioning is not yet implemented
var (
	verbose bool
	version = "main"
)

// GetField returns the value of the field with the given label
func (i OpItemList) GetField(label string) string {
	for _, field := range i {
		if field.Label == label {
			return field.Value
		}
	}
	return ""
}

// getVersion returns the version of the binary
func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok && version == "main" {
		version = info.Main.Version
	}
	return version
}

// PrintVersion prints the version of the binary
func PrintVersion() {
	if verbose {
		fmt.Fprintf(os.Stderr, "git-credential-1password %s\n", getVersion())
	}
}

// opGetItem runs "op item get --format json" command with the given name
func opGetItem(n string) (OpItemList, error) {
	// --fields username,password limits the output to only username and password
	opItemGet := exec.Command("op", "item", "get", "--format", "json", "--fields", "username,password", n)

	opItemRaw, err := opItemGet.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("opItemGet failed with %s\n%+s", err, opItemRaw)
	}

	// marhsal the raw output to OpItem struct
	var opItem OpItemList
	if err = json.Unmarshal(opItemRaw, &opItem); err != nil {
		return nil, fmt.Errorf("json.Unmarshal() failed with %s", err)
	}
	return opItem, nil
}

// ReadLines reads the input from stdin and returns a map of key value pairs
func ReadLines() (inputs map[string]string) {
	inputs = make(map[string]string)
	// create stdin reader
	reader := bufio.NewReader(os.Stdin)

	for {
		// line by line read from stdin
		line, _ := reader.ReadString('\n')

		// if the line is empty, break the loop
		if line == "" || line == "\n" {
			break
		}

		// create a slice of strings by splitting the line
		parts := strings.SplitN(line, "=", 2)

		// see if this can a key value pair
		if len(parts) == 2 {
			inputs[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		} else {
			log.Fatalf("Invalid input: %s", line)
		}
	}
	return inputs
}

func main() {
	flag.Usage = func() {
		PrintVersion()
		fmt.Fprintln(os.Stderr, "usage: git credential-1password [<options>] <action>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Actions:")
		fmt.Fprintln(os.Stderr, "  get            Generate credential [called by Git]")
		fmt.Fprintln(os.Stderr, "  store          Store credential [called by Git]")
		fmt.Fprintln(os.Stderr, "  erase          Erase credential [called by Git]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "See also https://github.com/ethrgeist/git-credential-1password")
	}
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	// git provides argument via stdin
	// ref: https://git-scm.com/docs/gitcredentials
	switch args[0] {
	case "get":
		// git sends the input to stdin
		gitInputs := ReadLines()

		// check if the host field is present in the input
		if _, ok := gitInputs["host"]; !ok {
			log.Fatalf("host is missing in the input")
		}

		// run "op item get --format json" command with the host value
		// this can only get, no other operations are allowed
		opItem, err := opGetItem(gitInputs["host"])
		if err != nil {
			log.Fatal(err)
		}

		// feed the username and password to git
		username := opItem.GetField("username")
		password := opItem.GetField("password")
		if username == "" || password == "" {
			log.Fatalf("username or password is empty, is the item named correctly?")
		}
		fmt.Printf("username=%s\n", username)
		fmt.Printf("password=%s\n", password)
	case "store":
		gitInputs := ReadLines()

		item, _ := opGetItem(gitInputs["host"])
		if item == nil {
			// run "op create item" command with the host value
			// this can only get, no other operations are allowed
			opCreate, err := exec.Command("op", "item", "create", "--category=Login", "--title="+gitInputs["host"], "--url="+gitInputs["protocol"]+"://"+gitInputs["host"], "username="+gitInputs["username"], "password="+gitInputs["password"]).CombinedOutput()
			if err != nil {
				log.Fatalf("op create item failed with %s %s", err, opCreate)
			}
		}
	case "erase":
		gitInputs := ReadLines()
		// run "op delete item" command with the host value
		exec.Command("op", "item", "delete", gitInputs["host"]).CombinedOutput()
	default:
		// unknown argument
		log.Fatalf("It doesn't look like anything to me. (Unknown argument: %s)\n", args[0])
	}
}

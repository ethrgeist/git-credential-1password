package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
)

// versioning is not yet implemented
var version = "main"

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
	fmt.Fprintf(os.Stderr, "git-credential-1password %s\n", getVersion())
}

func main() {
	accountFlag := flag.String("account", "", "1Password account")
	vaultFlag := flag.String("vault", "", "1Password vault")
	flag.StringVar(&prefix, "prefix", "", "1Password item name prefix")
	flag.StringVar(&tags, "tags", "git-credentials", "1Password item tags")
	versionFlag := flag.Bool("version", false, "Print version")

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

	if *versionFlag {
		PrintVersion()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	// set global variables based on flags
	if *accountFlag != "" {
		opItemFlags = append(opItemFlags, "--account", *accountFlag)
	}
	if *vaultFlag != "" {
		opItemFlags = append(opItemFlags, "--vault", *vaultFlag)
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
		opItem, err := opGetItem(itemName(gitInputs["host"]))
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
	case "erase":
		gitInputs := ReadLines()
		// run "op delete item" command with the host value
		buildOpItemCommand("delete", itemName(gitInputs["host"])).Run()
	default:
		// unknown argument
		log.Fatalf("It doesn't look like anything to me. (Unknown argument: %s)\n", args[0])
	}
}

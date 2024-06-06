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
		getCommand()
	case "store":
		storeCommand()
	case "erase":
		eraseCommand()
	default:
		// unknown argument
		log.Fatalf("It doesn't look like anything to me. (Unknown argument: %s)\n", args[0])
	}
}

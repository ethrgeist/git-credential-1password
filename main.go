package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/ethrgeist/git-credential-1password/internal"
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
	flag.StringVar(&internal.Account, "account", "my", "1Password account")
	flag.StringVar(&internal.Vault, "vault", "Private", "1Password vault")
	flag.StringVar(&internal.Prefix, "prefix", "", "1Password item name prefix")
	flag.StringVar(&internal.UsernameField, "username-field", "username", "What field to use for the username")
	flag.StringVar(&internal.PasswordField, "password-field", "password", "What field to use for the password")
	flag.BoolVar(&internal.AllowErase, "erase", false, "Allow erasing credentials")
	flag.StringVar(&internal.OpPath, "op-path", "", "Path to the op binary")
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

	if internal.UsernameField == "" || internal.PasswordField == "" {
		log.Fatalf("username and password field must be set")
		flag.Usage()
		os.Exit(2)
	}

	// set op cli parameters based on flags
	internal.OpItemFlags = append(internal.OpItemFlags, "--account", internal.Account)
	internal.OpItemFlags = append(internal.OpItemFlags, "--vault", internal.Vault)

	// git provides argument via stdin
	// ref: https://git-scm.com/docs/gitcredentials
	switch args[0] {
	case "get":
		internal.GetCommand()
	case "store":
		internal.StoreCommand()
	case "erase":
		if !internal.AllowErase {
			log.Fatalf("To enable erasing credentials, use the -erase true flag")
			flag.Usage()
			os.Exit(2)
		}
		internal.EraseCommand()
	default:
		// unknown argument
		log.Fatalf("It doesn't look like anything to me. (Unknown argument: %s)\n", args[0])
	}
}

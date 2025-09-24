package internal

import (
	"bufio"
	"log"
	"os"
	"strings"
)

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

		// split the line by the first '=' and create a key value pair
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			log.Fatalf("Invalid input: %s", line)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		inputs[key] = val
	}
	return inputs
}

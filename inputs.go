package main

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

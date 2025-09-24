package internal

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"
)

// ReadLines reads the input from stdin and returns a map of key value pairs
func ReadLines() (inputs map[string]string) {
	inputs = make(map[string]string)
	// create stdin reader
	r := bufio.NewReader(os.Stdin)
	w := bufio.NewWriter(os.Stdout)
	defer w.Flush()

	for {
		// line by line read from stdin
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
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

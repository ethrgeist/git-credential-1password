package internal

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ReadLines reads the input from a reader and returns a map of key value pairs.
// Input follows the git credential helper protocol: key=value lines terminated by
// a blank line or EOF. Values are taken literally (only the line ending is
// removed), so passwords with leading or trailing spaces survive.
func ReadLines(r io.Reader) (map[string]string, error) {
	inputs := make(map[string]string)
	br := bufio.NewReader(r)

	for {
		line, readErr := br.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid input: %s", line)
		}
		inputs[strings.TrimSpace(key)] = val

		if readErr == io.EOF {
			break
		}
	}
	return inputs, nil
}

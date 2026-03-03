package internal

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ReadLines reads the input from a reader and returns a map of key value pairs.
// Input follows the git credential helper protocol: key=value lines terminated by
// a blank line or EOF.
func ReadLines(r io.Reader) (map[string]string, error) {
	inputs := make(map[string]string)
	br := bufio.NewReader(r)

	for {
		line, err := br.ReadString('\n')
		if err == io.EOF {
			// Process any remaining content before EOF
			line = strings.TrimRight(line, "\r\n")
			if line != "" {
				key, val, ok := strings.Cut(line, "=")
				if !ok {
					return nil, fmt.Errorf("invalid input: %s", line)
				}
				inputs[strings.TrimSpace(key)] = strings.TrimSpace(val)
			}
			break
		}
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid input: %s", line)
		}
		inputs[strings.TrimSpace(key)] = strings.TrimSpace(val)
	}
	return inputs, nil
}

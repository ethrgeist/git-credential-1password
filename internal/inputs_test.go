package internal

import (
	"strings"
	"testing"
)

func TestReadLines(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "valid input with blank line terminator",
			input: "protocol=https\nhost=github.com\n\n",
			want: map[string]string{
				"protocol": "https",
				"host":     "github.com",
			},
		},
		{
			name:  "valid input EOF terminated",
			input: "protocol=https\nhost=github.com",
			want: map[string]string{
				"protocol": "https",
				"host":     "github.com",
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "single blank line",
			input: "\n",
			want:  map[string]string{},
		},
		{
			name:    "malformed line no equals",
			input:   "invalid-line\n",
			wantErr: true,
		},
		{
			name:    "malformed line no equals EOF",
			input:   "invalid-line",
			wantErr: true,
		},
		{
			name:  "whitespace trimming",
			input: " protocol = https \n host = github.com \n\n",
			want: map[string]string{
				"protocol": "https",
				"host":     "github.com",
			},
		},
		{
			name:  "value with equals sign",
			input: "password=my=password\n\n",
			want: map[string]string{
				"password": "my=password",
			},
		},
		{
			name:  "CRLF line endings",
			input: "protocol=https\r\nhost=github.com\r\n\r\n",
			want: map[string]string{
				"protocol": "https",
				"host":     "github.com",
			},
		},
		{
			name:  "full credential block",
			input: "protocol=https\nhost=github.com\nusername=alice\npassword=secret\n\n",
			want: map[string]string{
				"protocol": "https",
				"host":     "github.com",
				"username": "alice",
				"password": "secret",
			},
		},
		{
			name:  "single key-value EOF",
			input: "host=example.com",
			want: map[string]string{
				"host": "example.com",
			},
		},
		{
			name:  "empty value",
			input: "host=\n\n",
			want: map[string]string{
				"host": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadLines(strings.NewReader(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries, want %d: %v", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("got[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

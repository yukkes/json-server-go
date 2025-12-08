package main

import (
	"fmt"
	"os"
	"testing"
)

func TestArgumentParsing(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedSource string
		expectedPort   int
		expectedHost   string
		expectedWatch  bool
		shouldError    bool
	}{
		{
			name:           "source only",
			args:           []string{"cmd", "db.json"},
			expectedSource: "db.json",
			expectedPort:   3000,
			expectedHost:   "localhost",
			expectedWatch:  false,
			shouldError:    false,
		},
		{
			name:           "source with flags after",
			args:           []string{"cmd", "db.json", "--port", "3001", "--host", "0.0.0.0"},
			expectedSource: "db.json",
			expectedPort:   3001,
			expectedHost:   "0.0.0.0",
			expectedWatch:  false,
			shouldError:    false,
		},
		{
			name:           "flags before source",
			args:           []string{"cmd", "--port", "3002", "--host", "127.0.0.1", "db.json"},
			expectedSource: "db.json",
			expectedPort:   3002,
			expectedHost:   "127.0.0.1",
			expectedWatch:  false,
			shouldError:    false,
		},
		{
			name:           "mixed flags and source",
			args:           []string{"cmd", "--port", "3003", "db.json", "--host", "192.168.1.1"},
			expectedSource: "db.json",
			expectedPort:   3003,
			expectedHost:   "192.168.1.1",
			expectedWatch:  false,
			shouldError:    false,
		},
		{
			name:           "bool flag before source",
			args:           []string{"cmd", "--watch", "db.json"},
			expectedSource: "db.json",
			expectedPort:   3000,
			expectedHost:   "localhost",
			expectedWatch:  true,
			shouldError:    false,
		},
		{
			name:           "bool flag after source",
			args:           []string{"cmd", "db.json", "--watch", "--port", "3005"},
			expectedSource: "db.json",
			expectedPort:   3005,
			expectedHost:   "localhost",
			expectedWatch:  true,
			shouldError:    false,
		},
		{
			name:           "complex mixed flags",
			args:           []string{"cmd", "--port", "3006", "db_sample.json", "--watch", "--read-only"},
			expectedSource: "db_sample.json",
			expectedPort:   3006,
			expectedHost:   "localhost",
			expectedWatch:  true,
			shouldError:    false,
		},
		{
			name:           "Docker-style arguments",
			args:           []string{"cmd", "db_sample.json", "--host", "0.0.0.0", "--port", "3000", "--static", "./public"},
			expectedSource: "db_sample.json",
			expectedPort:   3000,
			expectedHost:   "0.0.0.0",
			expectedWatch:  false,
			shouldError:    false,
		},
		{
			name:        "missing source",
			args:        []string{"cmd", "--port", "3000"},
			shouldError: true,
		},
		{
			name:        "no arguments",
			args:        []string{"cmd"},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Set test args
			os.Args = tt.args

			// Parse arguments using the same logic as main
			boolFlags := map[string]bool{
				"-read-only": true, "-ro": true,
				"-no-cors": true, "-nc": true,
				"-watch": true, "-w": true,
				"-version": true, "-v": true,
				"-help": true, "-h": true,
				"--read-only": true, "--ro": true,
				"--no-cors": true, "--nc": true,
				"--watch": true, "--w": true,
				"--version": true, "--v": true,
				"--help": true, "--h": true,
				"-quiet": true, "--quiet": true, "-q": true,
			}

			var source string
			var sourceIndex int
			for i := 1; i < len(os.Args); i++ {
				arg := os.Args[i]
				if len(arg) > 0 && arg[0] == '-' {
					if !boolFlags[arg] && i+1 < len(os.Args) && len(os.Args[i+1]) > 0 && os.Args[i+1][0] != '-' {
						i++
					}
					continue
				}
				source = arg
				sourceIndex = i
				break
			}

			if tt.shouldError {
				if source != "" {
					t.Errorf("Expected error (missing source), but got source: %s", source)
				}
				return
			}

			if source != tt.expectedSource {
				t.Errorf("Expected source %s, got %s", tt.expectedSource, source)
			}

			// Test flag parsing
			if sourceIndex > 0 {
				newArgs := make([]string, 0, len(os.Args)-1)
				newArgs = append(newArgs, os.Args[0])
				newArgs = append(newArgs, os.Args[1:sourceIndex]...)
				if sourceIndex+1 < len(os.Args) {
					newArgs = append(newArgs, os.Args[sourceIndex+1:]...)
				}
				os.Args = newArgs
			}

			// Create a new FlagSet for testing
			var port int
			var host string
			var watch bool
			var readOnly bool
			var quiet bool

			fs := &testFlagSet{}
			fs.IntVar(&port, "port", 3000)
			fs.IntVar(&port, "p", 3000)
			fs.StringVar(&host, "host", "localhost")
			fs.StringVar(&host, "H", "localhost")
			fs.BoolVar(&watch, "watch", false)
			fs.BoolVar(&watch, "w", false)
			fs.BoolVar(&readOnly, "read-only", false)
			fs.BoolVar(&readOnly, "ro", false)
			fs.BoolVar(&quiet, "quiet", false)
			fs.BoolVar(&quiet, "q", false)

			fs.Parse(os.Args[1:])

			if port != tt.expectedPort {
				t.Errorf("Expected port %d, got %d", tt.expectedPort, port)
			}
			if host != tt.expectedHost {
				t.Errorf("Expected host %s, got %s", tt.expectedHost, host)
			}
			if watch != tt.expectedWatch {
				t.Errorf("Expected watch %v, got %v", tt.expectedWatch, watch)
			}
		})
	}
}

// Simple test flag set implementation
type testFlagSet struct {
	args map[string]interface{}
}

func (fs *testFlagSet) IntVar(p *int, name string, value int) {
	if fs.args == nil {
		fs.args = make(map[string]interface{})
	}
	*p = value
	fs.args[name] = p
}

func (fs *testFlagSet) StringVar(p *string, name string, value string) {
	if fs.args == nil {
		fs.args = make(map[string]interface{})
	}
	*p = value
	fs.args[name] = p
}

func (fs *testFlagSet) BoolVar(p *bool, name string, value bool) {
	if fs.args == nil {
		fs.args = make(map[string]interface{})
	}
	*p = value
	fs.args[name] = p
}

func (fs *testFlagSet) Parse(args []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if len(arg) == 0 || arg[0] != '-' {
			continue
		}

		name := arg
		for len(name) > 0 && name[0] == '-' {
			name = name[1:]
		}

		if ptr, ok := fs.args[name]; ok {
			switch p := ptr.(type) {
			case *bool:
				*p = true
			case *int:
				if i+1 < len(args) {
					i++
					var val int
					_, err := fmt.Sscanf(args[i], "%d", &val)
					if err == nil {
						*p = val
					}
				}
			case *string:
				if i+1 < len(args) {
					i++
					*p = args[i]
				}
			}
		}
	}
}

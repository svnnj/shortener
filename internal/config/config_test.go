package config

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	tests := []struct {
		name              string
		envVars           map[string]string
		flagVars          map[string]string
		defVars           map[string]string
		wantServerAddress string
		wantBaseURL       string
	}{
		{
			name: "env values",
			envVars: map[string]string{
				"SERVER_ADDRESS": "testhostenv",
				"BASE_URL":       "testurlenv",
			},
			flagVars: map[string]string{
				"a": "testhostflag",
				"b": "testurlflag",
			},
			defVars: map[string]string{
				"a": "localhost:8080",
				"b": "http://localhost:8080",
			},
			wantServerAddress: "testhostenv",
			wantBaseURL:       "testurlenv",
		},
		{
			name: "flag values",
			envVars: map[string]string{
				"SERVER_ADDRESS": "",
				"BASE_URL":       "",
			},
			flagVars: map[string]string{
				"a": "testhostflag",
				"b": "testurlflag",
			},
			defVars: map[string]string{
				"a": "localhost:8080",
				"b": "http://localhost:8080",
			},
			wantServerAddress: "testhostflag",
			wantBaseURL:       "testurlflag",
		},
		{
			name: "def values",
			envVars: map[string]string{
				"SERVER_ADDRESS": "",
				"BASE_URL":       "",
			},
			flagVars: map[string]string{},
			defVars: map[string]string{
				"a": "localhost:8080",
				"b": "http://localhost:8080",
			},
			wantServerAddress: "localhost:8080",
			wantBaseURL:       "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			args := []string{os.Args[0]}
			oldArgs := os.Args
			for k, v := range tt.flagVars {
				args = append(args, fmt.Sprintf(`--%s=%s`, k, v))
			}
			os.Args = args
			cfg := Get()
			os.Args = oldArgs
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			assert.Equal(t, tt.wantServerAddress, cfg.ServerAddress)
			assert.Equal(t, tt.wantBaseURL, cfg.BaseURL)
		})
	}
}

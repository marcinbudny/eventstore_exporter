package config

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedConfig Config
		errorExpected  bool
	}{
		{
			name: "no parameters specified reults in defaults",
			args: []string{},
			expectedConfig: Config{
				Timeout:                   time.Duration(8 * time.Second),
				Port:                      9448,
				Verbose:                   false,
				InsecureSkipVerify:        false,
				EventStoreURL:             "http://localhost:2113",
				EventStoreUser:            "",
				EventStorePassword:        "",
				EnableParkedMessagesStats: false,
				Streams:                   []string{},
				StreamsSeparator:          ",",
				EnableTCPConnectionStats:  false,
			},
		},
		{
			name: "all parameters specified",
			args: []string{
				"-timeout=20s",
				"-port=1231",
				"-verbose=true",
				"-insecure-skip-verify=true",
				"-eventstore-url=https://somewhere",
				"-eventstore-password=password",
				"-eventstore-user=user",
				"-enable-parked-messages-stats=true",
				"-streams=$all;my-stream;my-other-stream",
				"-streams-separator=;",
				"-enable-tcp-connection-stats=true",
			},
			expectedConfig: Config{
				Timeout:                   time.Duration(20 * time.Second),
				Port:                      1231,
				Verbose:                   true,
				InsecureSkipVerify:        true,
				EventStoreURL:             "https://somewhere",
				EventStoreUser:            "user",
				EventStorePassword:        "password",
				EnableParkedMessagesStats: true,
				Streams:                   []string{"$all", "my-stream", "my-other-stream"},
				StreamsSeparator:          ";",
				EnableTCPConnectionStats:  true,
			},
		},
		{
			name: "error on user name only",
			args: []string{
				"-eventstore-user=user",
			},
			errorExpected: true,
		},
		{
			name: "error on password only",
			args: []string{
				"-eventstore-password=password",
			},
			errorExpected: true,
		},
		{
			name: "error on streams separator",
			args: []string{
				"-streams-separator=test",
			},
			errorExpected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := Load(test.args, true)
			if err != nil && !test.errorExpected {
				t.Fatalf("uexpected error: %v", err)
			} else if err == nil && test.errorExpected {
				t.Fatal("expected error, but got nil")
			}
			if !test.errorExpected {
				if diff := cmp.Diff(*cfg, test.expectedConfig); diff != "" {
					t.Errorf("wrong config returned, diff: %v", diff)
				}
			}
		})
	}
}

func TestLoadConfigFromEnvironment(t *testing.T) {
	t.Setenv("TIMEOUT", "20s")
	t.Setenv("PORT", "1231")
	t.Setenv("INSECURE_SKIP_VERIFY", "true")
	t.Setenv("VERBOSE", "true")
	t.Setenv("EVENTSTORE_URL", "https://somewhere")
	t.Setenv("EVENTSTORE_USER", "user")
	t.Setenv("EVENTSTORE_PASSWORD", "password")
	t.Setenv("ENABLE_PARKED_MESSAGES_STATS", "true")
	t.Setenv("STREAMS", "$all;my-stream;my-other-stream")
	t.Setenv("STREAMS_SEPARATOR", ";")
	t.Setenv("ENABLE_TCP_CONNECTION_STATS", "true")

	expectedConfig := Config{
		Timeout:                   time.Duration(20 * time.Second),
		Port:                      1231,
		Verbose:                   true,
		InsecureSkipVerify:        true,
		EventStoreURL:             "https://somewhere",
		EventStoreUser:            "user",
		EventStorePassword:        "password",
		EnableParkedMessagesStats: true,
		Streams:                   []string{"$all", "my-stream", "my-other-stream"},
		StreamsSeparator:          ";",
		EnableTCPConnectionStats:  true,
	}

	if cfg, err := Load([]string{}, true); err == nil {
		if diff := cmp.Diff(*cfg, expectedConfig); diff != "" {
			t.Errorf("wrong config returned, diff: %v", diff)
		}
	} else {
		t.Errorf("unexpected error %v", err)
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	args := []string{"-config=sample_config"}

	expectedConfig := Config{
		Timeout:                   time.Duration(20 * time.Second),
		Port:                      1231,
		Verbose:                   true,
		InsecureSkipVerify:        true,
		EventStoreURL:             "https://somewhere_else",
		EventStoreUser:            "user",
		EventStorePassword:        "password",
		EnableParkedMessagesStats: true,
		Streams:                   []string{"$all", "my-test-stream", "my-other-stream"},
		StreamsSeparator:          "|",
		EnableTCPConnectionStats:  true,
	}

	if cfg, err := Load(args, true); err == nil {
		if diff := cmp.Diff(*cfg, expectedConfig); diff != "" {
			t.Errorf("wrong config returned, diff: %v", diff)
		}
	} else {
		t.Errorf("unexpected error %v", err)
	}
}

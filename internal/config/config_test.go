package config

import (
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	defer clearEnvironment()

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
				Timeout:                   time.Duration(10 * time.Second),
				Port:                      9448,
				Verbose:                   false,
				InsecureSkipVerify:        false,
				EventStoreURL:             "http://localhost:2113",
				EventStoreUser:            "",
				EventStorePassword:        "",
				ClusterMode:               "cluster",
				EnableParkedMessagesStats: false,
				Streams:                   []string{},
				StreamsSeparator:          ",",
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
				"-cluster-mode=single",
				"-enable-parked-messages-stats=true",
				"-streams=$all;my-stream;my-other-stream",
				"-streams-separator=;",
			},
			expectedConfig: Config{
				Timeout:                   time.Duration(20 * time.Second),
				Port:                      1231,
				Verbose:                   true,
				InsecureSkipVerify:        true,
				EventStoreURL:             "https://somewhere",
				EventStoreUser:            "user",
				EventStorePassword:        "password",
				ClusterMode:               "single",
				EnableParkedMessagesStats: true,
				Streams:                   []string{"$all", "my-stream", "my-other-stream"},
				StreamsSeparator:          ";",
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
			name: "error on invalid cluster mode",
			args: []string{
				"-cluster-mode=test",
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
	defer clearEnvironment()

	os.Setenv("TIMEOUT", "20s")
	os.Setenv("PORT", "1231")
	os.Setenv("INSECURE_SKIP_VERIFY", "true")
	os.Setenv("VERBOSE", "true")
	os.Setenv("EVENTSTORE_URL", "https://somewhere")
	os.Setenv("EVENTSTORE_USER", "user")
	os.Setenv("EVENTSTORE_PASSWORD", "password")
	os.Setenv("CLUSTER_MODE", "single")
	os.Setenv("ENABLE_PARKED_MESSAGES_STATS", "true")
	os.Setenv("STREAMS", "$all;my-stream;my-other-stream")
	os.Setenv("STREAMS_SEPARATOR", ";")

	expectedConfig := Config{
		Timeout:                   time.Duration(20 * time.Second),
		Port:                      1231,
		Verbose:                   true,
		InsecureSkipVerify:        true,
		EventStoreURL:             "https://somewhere",
		EventStoreUser:            "user",
		EventStorePassword:        "password",
		ClusterMode:               "single",
		EnableParkedMessagesStats: true,
		Streams:                   []string{"$all", "my-stream", "my-other-stream"},
		StreamsSeparator:          ";",
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
	defer clearEnvironment()

	args := []string{"-config=sample_config"}

	expectedConfig := Config{
		Timeout:                   time.Duration(20 * time.Second),
		Port:                      1231,
		Verbose:                   true,
		InsecureSkipVerify:        true,
		EventStoreURL:             "https://somewhere_else",
		EventStoreUser:            "user",
		EventStorePassword:        "password",
		ClusterMode:               "single",
		EnableParkedMessagesStats: true,
		Streams:                   []string{"$all", "my-test-stream", "my-other-stream"},
		StreamsSeparator:          "|",
	}

	if cfg, err := Load(args, true); err == nil {
		if diff := cmp.Diff(*cfg, expectedConfig); diff != "" {
			t.Errorf("wrong config returned, diff: %v", diff)
		}
	} else {
		t.Errorf("unexpected error %v", err)
	}
}

func clearEnvironment() {
	os.Unsetenv("TIMEOUT")
	os.Unsetenv("PORT")
	os.Unsetenv("INSECURE_SKIP_VERIFY")
	os.Unsetenv("VERBOSE")
	os.Unsetenv("EVENTSTORE_URL")
	os.Unsetenv("EVENTSTORE_USER")
	os.Unsetenv("EVENTSTORE_PASSWORD")
	os.Unsetenv("CLUSTER_MODE")
	os.Unsetenv("ENABLE_PARKED_MESSAGES_STATS")
	os.Unsetenv("STREAMS")
	os.Unsetenv("STREAMS_SEPARATOR")
}

package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "omil-config-test-*")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tmpDir)

	testCases := []struct {
		name           string
		configContent  string
		configPath     string
		expectedConfig *configStruct
		expectError    bool
		setupFunc      func(t *testing.T) // Additional setup if needed
	}{
		{
			name: "valid default config",
			configContent: `
Hostname: default-host
InfluxDBv2:
  Addr: http://localhost:8086
  Org: default-org
  Bucket: default-bucket
  Token: default-token
Targets:
  - Host: localhost
    Name: local
`,
			expectedConfig: &configStruct{
				Hostname: "default-host",
				InfluxDBv2: struct {
					Addr   string `yaml:"Addr"`
					Org    string `yaml:"Org"`
					Bucket string `yaml:"Bucket"`
					Token  string `yaml:"Token"`
				}{
					Addr:   "http://localhost:8086",
					Org:    "default-org",
					Bucket: "default-bucket",
					Token:  "default-token",
				},
				Targets: []Target{
					{
						Host: "localhost",
						Name: "local",
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid custom config with multiple targets",
			configContent: `
Hostname: test-host
InfluxDBv2:
  Addr: http://localhost:8086
  Org: test-org
  Bucket: test-bucket
  Token: test-token
Targets:
  - Host: google.com
    Name: google
  - Host: github.com
    Name: github
`,
			expectedConfig: &configStruct{
				Hostname: "test-host",
				InfluxDBv2: struct {
					Addr   string `yaml:"Addr"`
					Org    string `yaml:"Org"`
					Bucket string `yaml:"Bucket"`
					Token  string `yaml:"Token"`
				}{
					Addr:   "http://localhost:8086",
					Org:    "test-org",
					Bucket: "test-bucket",
					Token:  "test-token",
				},
				Targets: []Target{
					{
						Host: "google.com",
						Name: "google",
					},
					{
						Host: "github.com",
						Name: "github",
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid yaml syntax",
			configContent: `
Hostname: test-host
InfluxDBv2:
  Addr: http://localhost:8086
  Token: test-token
  Org: test-org
  Bucket: test-bucket
  : invalid-key  # Invalid YAML - empty key
Targets:
- Host: google.com
  Name: google
  : another-invalid-key
`,
			expectError: true,
		},
		{
			name:        "nonexistent config file",
			configPath:  "nonexistent.yml",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset package-level variables
			config = nil
			configFilePath = ""
			initConfigOnce = sync.Once{}

			// Create test config file if content is provided
			var testConfigPath string
			if tc.configContent != "" {
				testConfigPath = filepath.Join(tmpDir, tc.name+".yml")
				err := os.WriteFile(testConfigPath, []byte(tc.configContent), 0644)
				require.NoError(t, err, "Failed to write test config file")
			} else if tc.configPath != "" {
				testConfigPath = tc.configPath
			}

			if testConfigPath != "" {
				SetConfigFilePath(testConfigPath)
			}

			// Create test logger and capture its output
			logger, buf := testLogger()
			logger.SetLevel(logrus.DebugLevel)
			logger.ExitFunc = func(int) { } // Prevent logger from calling os.Exit
			origLogger := logrus.StandardLogger()

			// Replace the global logger
			logrus.SetOutput(logger.Out)
			logrus.SetLevel(logrus.DebugLevel)
			logrus.SetFormatter(&logrus.TextFormatter{
				DisableColors: true,
				FullTimestamp: true,
			})
			// Override the exit function for the global logger too
			logrus.StandardLogger().ExitFunc = func(int) { }
			defer func() {
				logrus.SetOutput(origLogger.Out)
				logrus.SetFormatter(origLogger.Formatter)
				logrus.SetLevel(origLogger.GetLevel())
				logrus.StandardLogger().ExitFunc = origLogger.ExitFunc
			}()

			t.Logf("Running test case: %s with config path: %s", tc.name, testConfigPath)

			// Save original os.Exit and restore it after test
			origExit := osExit
			defer func() { osExit = origExit }()

			exitCalled := false
			osExit = func(code int) {
				exitCalled = true
			}

			// Run test with timeout
			done := make(chan struct{})
			var cfg *configStruct
			go func() {
				defer close(done)
				defer func() {
					if r := recover(); r != nil {
						t.Logf("Recovered from panic: %v", r)
					}
				}()
				cfg = Config()
			}()

			select {
			case <-done:
				if tc.expectError {
					require.True(t, exitCalled, "Expected os.Exit to be called for invalid config")
					output := buf.String()
					t.Logf("Test output for %s: %q", tc.name, output)
					if !exitCalled {
						t.Errorf("Expected os.Exit to be called but it wasn't")
					}
					if !strings.Contains(output, "failed to load config") && !strings.Contains(output, "config file not found") {
						t.Errorf("Expected error message not found in output: %s", output)
					}
				} else {
					require.False(t, exitCalled, "os.Exit was called unexpectedly")
					require.NotNil(t, cfg, "Expected non-nil config")
					require.Equal(t, tc.expectedConfig.Hostname, cfg.Hostname,
						"Unexpected hostname")
					require.Equal(t, tc.expectedConfig.InfluxDBv2, cfg.InfluxDBv2,
						"Unexpected InfluxDBv2 config")
					require.Equal(t, len(tc.expectedConfig.Targets), len(cfg.Targets),
						"Unexpected number of targets")
					for i, target := range tc.expectedConfig.Targets {
						require.Equal(t, target, cfg.Targets[i],
							"Unexpected target at index %d", i)
					}
				}
			case <-time.After(time.Second):
				t.Error("Test timed out")
				t.Logf("Buffer contents at timeout: %q", buf.String())
			}
		})
	}
}

// testLogger creates a new logrus logger instance with a buffer for capturing output
func testLogger() (*logrus.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := logrus.New()
	logger.Out = &buf
	logger.Formatter = &logrus.TextFormatter{
		DisableColors:    true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	}
	return logger, &buf
}

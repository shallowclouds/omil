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
)

func TestConfig(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "omil-config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Reset package-level variables before each test
	config = nil
	configFilePath = ""
	initConfigOnce = sync.Once{}

	t.Run("default config path", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		// Create conf directory in temp dir
		confDir := filepath.Join(tmpDir, "conf")
		if err := os.MkdirAll(confDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Create default config file
		defaultConfigPath := filepath.Join(confDir, "config.yml")
		err := os.WriteFile(defaultConfigPath, []byte(`
Hostname: default-host
InfluxDBv2:
  Addr: http://localhost:8086
  Org: default-org
  Bucket: default-bucket
  Token: default-token
Targets:
  - Host: localhost
    Name: local
`), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Change working directory to temp dir
		origWd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		defer os.Chdir(origWd)

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal(err)
		}

		// Save original os.Exit and restore it after test
		origExit := osExit
		defer func() { osExit = origExit }()

		exitCalled := false
		osExit = func(code int) {
			exitCalled = true
		}

		cfg := Config()
		if exitCalled {
			t.Fatal("os.Exit was called unexpectedly")
		}
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		// Verify config parsing
		if cfg.Hostname != "default-host" {
			t.Errorf("expected hostname 'default-host', got %q", cfg.Hostname)
		}

		if cfg.InfluxDBv2.Addr != "http://localhost:8086" {
			t.Errorf("expected InfluxDB addr 'http://localhost:8086', got %q", cfg.InfluxDBv2.Addr)
		}

		if len(cfg.Targets) != 1 {
			t.Errorf("expected 1 target, got %d", len(cfg.Targets))
		}

		// Verify target
		if cfg.Targets[0].Host != "localhost" || cfg.Targets[0].Name != "local" {
			t.Errorf("unexpected target: %+v", cfg.Targets[0])
		}
	})

	t.Run("custom config path", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		// Create test config in temp dir
		testConfigPath := filepath.Join(tmpDir, "valid_config.yml")
		err := os.WriteFile(testConfigPath, []byte(`
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
`), 0644)
		if err != nil {
			t.Fatal(err)
		}

		SetConfigFilePath(testConfigPath)

		// Save original os.Exit and restore it after test
		origExit := osExit
		defer func() { osExit = origExit }()

		exitCalled := false
		osExit = func(code int) {
			exitCalled = true
		}

		cfg := Config()
		if exitCalled {
			t.Fatal("os.Exit was called unexpectedly")
		}
		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		// Verify config parsing
		if cfg.Hostname != "test-host" {
			t.Errorf("expected hostname 'test-host', got %q", cfg.Hostname)
		}

		if cfg.InfluxDBv2.Addr != "http://localhost:8086" {
			t.Errorf("expected InfluxDB addr 'http://localhost:8086', got %q", cfg.InfluxDBv2.Addr)
		}

		if len(cfg.Targets) != 2 {
			t.Errorf("expected 2 targets, got %d", len(cfg.Targets))
		}

		// Verify first target
		if cfg.Targets[0].Host != "google.com" || cfg.Targets[0].Name != "google" {
			t.Errorf("unexpected first target: %+v", cfg.Targets[0])
		}

		// Verify second target
		if cfg.Targets[1].Host != "github.com" || cfg.Targets[1].Name != "github" {
			t.Errorf("unexpected second target: %+v", cfg.Targets[1])
		}
	})

	t.Run("invalid config file", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		// Create invalid config in temp dir
		testConfigPath := filepath.Join(tmpDir, "invalid_config.yml")
		err := os.WriteFile(testConfigPath, []byte(`
# Invalid YAML with basic syntax error
Hostname: test-host
InfluxDBv2:
  Addr: http://localhost:8086
  Token: test-token
  Org: test-org
  Bucket: test-bucket
  : invalid-key  # This is invalid YAML - key cannot be empty
Targets:
- Host: google.com
  Name: google
  : another-invalid-key  # This is also invalid YAML
`), 0644)
		if err != nil {
			t.Fatal(err)
		}

		SetConfigFilePath(testConfigPath)

		// Create test logger and capture its output BEFORE calling Config
		logger, buf := testLogger()
		logger.SetLevel(logrus.FatalLevel)
		origLogger := logrus.StandardLogger()

		// Replace the global logger and its formatter
		logrus.SetOutput(logger.Out)
		logrus.SetFormatter(&logrus.TextFormatter{
			DisableColors: true,
			FullTimestamp: true,
		})
		defer func() {
			logrus.SetOutput(origLogger.Out)
			logrus.SetFormatter(origLogger.Formatter)
		}()

		// Save original os.Exit and restore it after test
		origExit := osExit
		defer func() { osExit = origExit }()

		exitCalled := false
		osExit = func(code int) {
			exitCalled = true
			t.Logf("os.Exit called with code %d", code)
			// Print buffer contents at exit time
			t.Logf("Logger buffer at exit: %q", buf.String())
		}

		// Add debug logging
		t.Logf("Test starting with config file: %s", testConfigPath)
		if _, err := os.Stat(testConfigPath); err != nil {
			t.Logf("Config file status: %v", err)
		} else {
			t.Log("Config file exists")
		}

		// Call Config() in a goroutine since it will exit
		done := make(chan struct{})
		go func() {
			defer close(done)
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Recovered from panic: %v", r)
				}
			}()
			Config()
		}()

		// Wait for either exit to be called or timeout
		select {
		case <-done:
			output := buf.String()
			t.Logf("Final logger output: %q", output)
			if !exitCalled {
				t.Error("expected os.Exit to be called for invalid config")
			}
			expectedMsg := "failed to load config from file"
			if !strings.Contains(output, expectedMsg) {
				t.Errorf("expected error message to contain %q, got: %q", expectedMsg, output)
				// Print the output in a more readable format
				t.Logf("Logger output (line by line):")
				for i, line := range strings.Split(output, "\n") {
					t.Logf("Line %d: %q", i+1, line)
				}
			}
		case <-time.After(time.Second):
			t.Error("test timed out waiting for Config() to complete")
			t.Logf("Buffer contents at timeout: %q", buf.String())
		}
	})

	t.Run("nonexistent config file", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		nonexistentPath := filepath.Join(tmpDir, "nonexistent.yml")
		SetConfigFilePath(nonexistentPath)

		// Save original os.Exit and restore it after test
		origExit := osExit
		defer func() { osExit = origExit }()

		exitCalled := false
		osExit = func(code int) {
			exitCalled = true
		}

		// Call Config() in a goroutine since it will exit
		done := make(chan struct{})
		go func() {
			Config()
			close(done)
		}()

		// Wait for either exit to be called or timeout
		select {
		case <-done:
			if !exitCalled {
				t.Error("expected os.Exit to be called for nonexistent config")
			}
		case <-time.After(time.Second):
			t.Error("test timed out waiting for Config() to complete")
		}
	})
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

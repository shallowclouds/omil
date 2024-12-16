package config

import (
	"fmt"
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

		// Save original os.Exit and restore it after test
		origExit := osExit
		defer func() { osExit = origExit }()

		// Save original logrus.Fatal and restore it after test
		origLogFatal := logrus.Fatal
		defer func() { logrus.Fatal = origLogFatal }()

		exitCalled := false
		osExit = func(code int) {
			exitCalled = true
		}

		var loggedError string
		logrus.Fatal = func(args ...interface{}) {
			if len(args) > 0 {
				loggedError = fmt.Sprint(args...)
			}
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
				t.Error("expected os.Exit to be called for invalid config")
			}
			if !strings.Contains(loggedError, "failed to load config") {
				t.Errorf("expected error message to contain 'failed to load config', got: %q", loggedError)
			}
		case <-time.After(time.Second):
			t.Error("test timed out waiting for Config() to complete")
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

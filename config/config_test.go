package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	// Reset package-level variables before each test
	config = nil
	configFilePath = ""
	initConfigOnce = sync.Once{}

	t.Run("default config path", func(t *testing.T) {
		// Should try to load from default path conf/config.yml
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		cfg := Config()
		if cfg == nil {
			t.Error("expected non-nil config even with missing default config")
		}
	})

	t.Run("custom config path", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		testConfigPath := filepath.Join("testdata", "valid_config.yml")
		SetConfigFilePath(testConfigPath)

		cfg := Config()
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

		testConfigPath := filepath.Join("testdata", "invalid_config.yml")
		SetConfigFilePath(testConfigPath)

		// Config() calls log.Fatal on error, so we need to prevent that
		// Save original os.Exit and restore it after test
		originalOsExit := osExit
		defer func() { osExit = originalOsExit }()

		exitCalled := false
		osExit = func(code int) {
			exitCalled = true
		}

		Config()
		if !exitCalled {
			t.Error("expected os.Exit to be called for invalid config")
		}
	})

	t.Run("nonexistent config file", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		SetConfigFilePath("nonexistent.yml")

		// Config() calls log.Fatal on error, so we need to prevent that
		originalOsExit := osExit
		defer func() { osExit = originalOsExit }()

		exitCalled := false
		osExit = func(code int) {
			exitCalled = true
		}

		Config()
		if !exitCalled {
			t.Error("expected os.Exit to be called for nonexistent config")
		}
	})
}

// Override os.Exit for testing
var osExit = os.Exit

package config

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	// Reset package-level variables before each test
	config = nil
	configFilePath = ""
	initConfigOnce = sync.Once{}

	t.Run("default config path", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

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
				t.Error("expected os.Exit to be called when default config is missing")
			}
		case <-time.After(time.Second):
			t.Error("test timed out waiting for Config() to complete")
		}
	})

	t.Run("custom config path", func(t *testing.T) {
		// Reset package-level variables
		config = nil
		configFilePath = ""
		initConfigOnce = sync.Once{}

		// Get absolute path to test config
		pwd, err := filepath.Abs(".")
		if err != nil {
			t.Fatal(err)
		}
		testConfigPath := filepath.Join(pwd, "testdata", "valid_config.yml")
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

		// Get absolute path to test config
		pwd, err := filepath.Abs(".")
		if err != nil {
			t.Fatal(err)
		}
		testConfigPath := filepath.Join(pwd, "testdata", "invalid_config.yml")
		SetConfigFilePath(testConfigPath)

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
				t.Error("expected os.Exit to be called for invalid config")
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

		SetConfigFilePath("nonexistent.yml")

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

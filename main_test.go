package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/shallowclouds/omil/metric"
	"github.com/urfave/cli/v2"
)

// mockMetricClient implements metric.Client interface for testing
type mockMetricClient struct {
	metric.Client
	points  []struct {
		name   string
		fields map[string]interface{}
		tags   map[string]string
	}
	exitErr error
}

func (m *mockMetricClient) Send(point struct {
	name   string
	fields map[string]interface{}
	tags   map[string]string
}) error {
	m.points = append(m.points, point)
	return nil
}

func (m *mockMetricClient) Exit() error {
	return m.exitErr
}

// TestMainAction_ConfigLoading tests the config loading functionality
func TestMainAction_ConfigLoading(t *testing.T) {
	// Create a temporary config file
	tmpConfig := `
influxdb_v2:
  addr: http://localhost:8086
  org: test-org
  bucket: test-bucket
  token: test-token
targets:
  - host: localhost
    interval: 1s
`
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(tmpConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp config: %v", err)
	}

	// Test with valid config
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("config", tmpFile.Name(), "doc")
	ctx := cli.NewContext(app, set, nil)

	err = mainAction(ctx)
	if err != nil {
		t.Errorf("mainAction() with valid config error = %v, want nil", err)
	}

	// Test with invalid config path
	set = flag.NewFlagSet("test", 0)
	set.String("config", "nonexistent.yml", "doc")
	ctx = cli.NewContext(app, set, nil)
	err = mainAction(ctx)
	if err == nil {
		t.Error("mainAction() with invalid config path error = nil, want error")
	}
}

// TestMainAction_MonitorCreation tests monitor creation with various targets
func TestMainAction_MonitorCreation(t *testing.T) {
	// Test with valid and invalid targets
	tmpConfig := `
influxdb_v2:
  addr: http://localhost:8086
  org: test-org
  bucket: test-bucket
  token: test-token
targets:
  - host: localhost
    interval: 1s
  - host: ""
    interval: 1s
  - host: invalid.host.local
    interval: 1s
`
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(tmpConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp config: %v", err)
	}

	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("config", tmpFile.Name(), "doc")
	ctx := cli.NewContext(app, set, nil)

	err = mainAction(ctx)
	if err != nil {
		t.Errorf("mainAction() with mixed targets error = %v, want nil", err)
	}
}

// TestMainAction_LoopError tests error handling in the monitor loop
func TestMainAction_LoopError(t *testing.T) {
	// Test with interrupt signal
	tmpConfig := `
influxdb_v2:
  addr: http://localhost:8086
  org: test-org
  bucket: test-bucket
  token: test-token
targets:
  - host: localhost
    interval: 1s
`
	tmpFile, err := os.CreateTemp("", "config-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp config: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(tmpConfig)); err != nil {
		t.Fatalf("Failed to write temp config: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp config: %v", err)
	}

	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	set.String("config", tmpFile.Name(), "doc")
	ctx := cli.NewContext(app, set, nil)

	// Simulate interrupt
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(os.Interrupt)
	}()

	err = mainAction(ctx)
	if err != nil {
		t.Errorf("mainAction() with interrupt error = %v, want nil", err)
	}

	// Test with other loop error
	ctx = cli.NewContext(app, set, nil)
	ctx.Context = context.WithValue(ctx.Context, "test_error", errors.New("test error"))

	err = mainAction(ctx)
	if err == nil {
		t.Error("mainAction() with loop error = nil, want error")
	}
}

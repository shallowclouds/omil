package main

import (
	"context"
	"flag"
	"testing"

	"github.com/shallowclouds/omil/loop"
	"github.com/urfave/cli/v2"
)

// mockMonitor implements loop.Monitor interface for testing
type mockMonitor struct {
	name     string
	startErr error
	stopErr  error
}

func (m *mockMonitor) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	<-ctx.Done()
	return ctx.Err()
}

func (m *mockMonitor) Stop() error {
	return m.stopErr
}

func (m *mockMonitor) Name() string {
	return m.name
}

// TestMainAction_MonitorLoop tests the monitor loop functionality
func TestMainAction_MonitorLoop(t *testing.T) {
	// Create a mock monitor
	mock := &mockMonitor{
		name: "test",
	}

	// Create CLI context
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)

	// Test normal operation
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(os.Interrupt)
	}()

	err := loop.Loop(ctx.Context, []loop.Monitor{mock})
	if err != loop.ErrInterrupt {
		t.Errorf("Loop() error = %v, want %v", err, loop.ErrInterrupt)
	}

	// Test monitor error
	mock.startErr = errors.New("start error")
	err = loop.Loop(ctx.Context, []loop.Monitor{mock})
	if err != nil {
		t.Errorf("Loop() with monitor error = %v, want nil", err)
	}
}

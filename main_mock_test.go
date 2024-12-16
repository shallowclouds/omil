package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/shallowclouds/omil/loop"
	"github.com/urfave/cli/v2"
)

// mockMonitor implements loop.Monitor interface for testing
type mockMonitor struct {
	name     string
	startErr error
	stopErr  error
	stopped  bool
}

func (m *mockMonitor) Start(ctx context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	<-ctx.Done()
	return ctx.Err()
}

func (m *mockMonitor) Stop() error {
	m.stopped = true
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

	// Create a context with timeout to prevent test from hanging
	ctxWithTimeout, cancel := context.WithTimeout(ctx.Context, 2*time.Second)
	defer cancel()

	// Test normal operation
	go func() {
		time.Sleep(100 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(os.Interrupt)
	}()

	err := loop.Loop(ctxWithTimeout, []loop.Monitor{mock})
	if err != loop.ErrInterrupt {
		t.Errorf("Loop() error = %v, want %v", err, loop.ErrInterrupt)
	}

	if !mock.stopped {
		t.Error("Monitor was not stopped after interrupt")
	}

	// Test monitor error
	mock = &mockMonitor{
		name:     "test",
		startErr: errors.New("start error"),
	}

	err = loop.Loop(ctxWithTimeout, []loop.Monitor{mock})
	if err != nil {
		t.Errorf("Loop() with monitor error = %v, want nil", err)
	}

	if !mock.stopped {
		t.Error("Monitor was not stopped after error")
	}
}

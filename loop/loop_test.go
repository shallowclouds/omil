package loop

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// mockMonitor implements a controllable monitor for testing
type mockMonitor struct {
	name       string
	startErr   error
	stopErr    error
	startChan  chan struct{}
	stopChan   chan struct{}
	stopped    bool
	mu         sync.RWMutex
	running    bool
	runningMu  sync.RWMutex
}

func newMockMonitor(name string) *mockMonitor {
	return &mockMonitor{
		name:      name,
		startChan: make(chan struct{}, 1),
		stopChan:  make(chan struct{}, 1),
	}
}

func (m *mockMonitor) Start(ctx context.Context) error {
	m.runningMu.Lock()
	if m.running {
		m.runningMu.Unlock()
		<-ctx.Done()
		return ctx.Err()
	}
	m.running = true
	m.runningMu.Unlock()

	defer func() {
		m.runningMu.Lock()
		m.running = false
		m.runningMu.Unlock()
	}()

	if m.startErr != nil {
		select {
		case m.startChan <- struct{}{}:
		default:
		}
		return m.startErr
	}

	select {
	case m.startChan <- struct{}{}:
	default:
	}

	select {
	case <-ctx.Done():
		time.Sleep(10 * time.Millisecond)
		return ctx.Err()
	}
}

func (m *mockMonitor) Stop() error {
	m.mu.Lock()
	m.stopped = true
	m.mu.Unlock()

	m.runningMu.Lock()
	m.running = false
	m.runningMu.Unlock()

	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopChan <- struct{}{}
	return nil
}

func (m *mockMonitor) Name() string {
	return m.name
}

// waitForStart waits for the monitor to start or timeout
func (m *mockMonitor) waitForStart(timeout time.Duration) error {
	select {
	case <-m.startChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for monitor to start")
	}
}

// waitForStop waits for the monitor to stop or timeout
func (m *mockMonitor) waitForStop(timeout time.Duration) error {
	select {
	case <-m.stopChan:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("timeout waiting for monitor to stop")
	}
}

// TestLoop_NormalOperation tests the normal operation of the Loop function
func TestLoop_NormalOperation(t *testing.T) {
	mock1 := newMockMonitor("test1")
	mock2 := newMockMonitor("test2")
	monitors := []Monitor{mock1, mock2}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Loop(ctx, monitors)
	}()

	// Verify monitors are started
	if err := mock1.waitForStart(time.Second); err != nil {
		t.Errorf("Monitor 1 failed to start: %v", err)
	}
	if err := mock2.waitForStart(time.Second); err != nil {
		t.Errorf("Monitor 2 failed to start: %v", err)
	}

	// Cancel context to trigger shutdown
	cancel()

	// Verify monitors are stopped
	if err := mock1.waitForStop(time.Second); err != nil {
		t.Errorf("Monitor 1 failed to stop: %v", err)
	}
	if err := mock2.waitForStop(time.Second); err != nil {
		t.Errorf("Monitor 2 failed to stop: %v", err)
	}

	// Wait for completion
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Loop() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Error("Loop() timeout waiting for completion")
	}
}

// TestLoop_MonitorFailure tests the behavior when monitors fail and restart
func TestLoop_MonitorFailure(t *testing.T) {
	mock := newMockMonitor("test")
	mock.startErr = fmt.Errorf("start error")
	monitors := []Monitor{mock}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Loop(ctx, monitors)
	}()

	// Monitor should attempt to start multiple times
	startAttempts := 0
	deadline := time.After(2 * restartInterval)
	for {
		select {
		case <-mock.startChan:
			startAttempts++
			if startAttempts >= 2 {
				goto done
			}
		case <-deadline:
			if startAttempts < 2 {
				t.Errorf("Expected multiple start attempts, got %d", startAttempts)
			}
			goto done
		}
	}
done:
	cancel()

	// Wait for completion
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Loop() error = %v, want nil", err)
		}
	case <-time.After(restartInterval):
		t.Error("Loop() timeout waiting for completion")
	}
}

// TestLoop_GracefulShutdown tests graceful shutdown via context cancellation
func TestLoop_GracefulShutdown(t *testing.T) {
	mock := newMockMonitor("test")
	monitors := []Monitor{mock}

	ctx, cancel := context.WithCancel(context.Background())

	errChan := make(chan error, 1)
	go func() {
		errChan <- Loop(ctx, monitors)
	}()

	// Wait for monitor to start
	if err := mock.waitForStart(time.Second); err != nil {
		t.Fatalf("Monitor failed to start: %v", err)
	}

	// Trigger shutdown
	cancel()

	// Verify monitor is stopped
	if err := mock.waitForStop(time.Second); err != nil {
		t.Errorf("Monitor failed to stop: %v", err)
	}

	// Wait for completion
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Loop() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Loop() timeout waiting for completion")
	}
}

// TestLoop_SignalInterrupt tests handling of interrupt signals
func TestLoop_SignalInterrupt(t *testing.T) {
	mock := newMockMonitor("test")
	monitors := []Monitor{mock}

	ctx := context.Background()

	errChan := make(chan error, 1)
	go func() {
		errChan <- Loop(ctx, monitors)
	}()

	// Wait for monitor to start
	if err := mock.waitForStart(time.Second); err != nil {
		t.Fatalf("Monitor failed to start: %v", err)
	}

	// Simulate interrupt signal once and wait for stop
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	// Send interrupt and immediately wait for stop to avoid multiple signals
	if err := p.Signal(os.Interrupt); err != nil {
		t.Fatalf("Failed to send interrupt signal: %v", err)
	}

	// Verify monitor is stopped
	if err := mock.waitForStop(time.Second); err != nil {
		t.Errorf("Monitor failed to stop: %v", err)
	}

	// Wait for completion with interrupt error
	select {
	case err := <-errChan:
		if err != ErrInterrupt {
			t.Errorf("Loop() error = %v, want %v", err, ErrInterrupt)
		}
	case <-time.After(2 * time.Second):
		t.Error("Loop() timeout waiting for completion")
	}
}

package metric

import (
	"sync"
	"time"
)

// MockClient implements Client interface for testing purposes
type MockClient struct {
	MetricCalls []struct {
		Name      string
		Timestamp time.Time
		Tags      map[string]string
		Value     map[string]interface{}
	}
	FlushCalled bool
	ExitCalled  bool

	MetricErr error
	FlushErr  error
	ExitErr   error

	// Async operation support
	metricChan chan struct {
		Name      string
		Timestamp time.Time
		Tags      map[string]string
		Value     map[string]interface{}
	}
	stopChan chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
}

// NewMockClient creates a new MockClient for testing
func NewMockClient() *MockClient {
	m := &MockClient{
		MetricCalls: make([]struct {
			Name      string
			Timestamp time.Time
			Tags      map[string]string
			Value     map[string]interface{}
		}, 0),
		metricChan: make(chan struct {
			Name      string
			Timestamp time.Time
			Tags      map[string]string
			Value     map[string]interface{}
		}, 100),
		stopChan: make(chan struct{}),
	}

	m.wg.Add(1)
	go m.processMetrics()
	return m
}

// processMetrics handles async metric processing
func (m *MockClient) processMetrics() {
	defer m.wg.Done()
	for {
		select {
		case metric := <-m.metricChan:
			m.mu.Lock()
			m.MetricCalls = append(m.MetricCalls, metric)
			m.mu.Unlock()
		case <-m.stopChan:
			return
		}
	}
}

// Metric sends the metric asynchronously
func (m *MockClient) Metric(name string, timestamp time.Time, tags map[string]string, value map[string]interface{}) {
	select {
	case m.metricChan <- struct {
		Name      string
		Timestamp time.Time
		Tags      map[string]string
		Value     map[string]interface{}
	}{name, timestamp, tags, value}:
	default:
		// Channel full, simulate dropping metrics like a real async client might
	}
}

// Flush records that Flush was called and returns any configured error
func (m *MockClient) Flush() error {
	m.FlushCalled = true
	return m.FlushErr
}

// Exit records that Exit was called, stops the metric processor, and returns any configured error
func (m *MockClient) Exit() error {
	m.ExitCalled = true
	close(m.stopChan)
	m.wg.Wait() // Wait for processor to finish
	return m.ExitErr
}

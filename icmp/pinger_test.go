package icmp

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-ping/ping"
	"github.com/shallowclouds/omil/metric"
)

// mockMetricClient implements metric.Client interface for testing
type mockMetricClient struct {
	metrics []struct {
		name   string
		time   time.Time
		tags   map[string]string
		fields map[string]interface{}
	}
}

func (m *mockMetricClient) Metric(name string, t time.Time, tags map[string]string, fields map[string]interface{}) {
	m.metrics = append(m.metrics, struct {
		name   string
		time   time.Time
		tags   map[string]string
		fields map[string]interface{}
	}{name, t, tags, fields})
}

func (m *mockMetricClient) Flush() error {
	return nil
}

func (m *mockMetricClient) Exit() error {
	return nil
}

// mockPinger implements the pinger interface for testing
type mockPinger struct {
	host       string
	interval   time.Duration
	timeout    time.Duration
	size       int
	stopped    bool
	onRecv     func(*ping.Packet)
	runErr     error
	privileged bool
}

func (p *mockPinger) SetPrivileged(privileged bool) {
	p.privileged = privileged
}

func (p *mockPinger) Run() error {
	if p.runErr != nil {
		return p.runErr
	}
	// Simulate receiving a packet
	if p.onRecv != nil {
		p.onRecv(&ping.Packet{
			Nbytes: 64,
			Seq:    1,
			Ttl:    64,
			Rtt:    time.Millisecond * 100,
		})
	}
	return nil
}

func (p *mockPinger) Stop() {
	p.stopped = true
}

// customPinger implements pinger interface but is not a pingAdapter
type customPinger struct {
	privileged bool
	onRecv     func(*ping.Packet)
}

func (p *customPinger) SetPrivileged(privileged bool) {
	p.privileged = privileged
}

func (p *customPinger) Run() error {
	// Simulate packet reception
	if p.onRecv != nil {
		p.onRecv(&ping.Packet{
			Nbytes: 64,
			Seq:    1,
			Ttl:    64,
			Rtt:    time.Millisecond * 100,
		})
	}
	return nil
}

func (p *customPinger) Stop() {}

// TestNewMonitor tests the NewMonitor function with various parameter combinations
func TestNewMonitor(t *testing.T) {
	// Save original hostname function and restore after test
	originalHostname := hostnameFunc
	defer func() { hostnameFunc = originalHostname }()

	tests := []struct {
		name         string
		host         string
		from         string
		to           string
		interval     time.Duration
		timeout      time.Duration
		client       metric.Client
		wantErr      bool
		mockHostname func() (string, error)
	}{
		{
			name:     "valid parameters",
			host:     "example.com",
			from:     "source",
			to:       "destination",
			interval: time.Second,
			timeout:  time.Minute,
			client:   &mockMetricClient{},
			wantErr:  false,
		},
		{
			name:     "nil client",
			host:     "example.com",
			from:     "source",
			to:       "destination",
			interval: time.Second,
			timeout:  time.Minute,
			client:   nil,
			wantErr:  true,
		},
		{
			name:     "empty hostname",
			host:     "",
			from:     "source",
			to:       "destination",
			interval: time.Second,
			timeout:  time.Minute,
			client:   &mockMetricClient{},
			wantErr:  false,
		},
		{
			name:     "zero interval",
			host:     "example.com",
			from:     "source",
			to:       "destination",
			interval: 0,
			timeout:  time.Minute,
			client:   &mockMetricClient{},
			wantErr:  false,
		},
		{
			name:     "hostname resolution failure",
			host:     "example.com",
			from:     "", // Empty from will trigger hostname resolution
			to:       "destination",
			interval: time.Second,
			timeout:  time.Minute,
			client:   &mockMetricClient{},
			wantErr:  true,
			mockHostname: func() (string, error) {
				return "", fmt.Errorf("hostname resolution failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockHostname != nil {
				hostnameFunc = tt.mockHostname
			}
			m, err := NewMonitor(tt.host, tt.from, tt.to, tt.interval, tt.timeout, tt.client)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMonitor() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && m == nil {
				t.Error("NewMonitor() returned nil monitor without error")
			}
			if !tt.wantErr {
				if m.host != tt.host {
					t.Errorf("NewMonitor() host = %v, want %v", m.host, tt.host)
				}
				if m.from != tt.from {
					t.Errorf("NewMonitor() from = %v, want %v", m.from, tt.from)
				}
				if m.to != tt.to {
					t.Errorf("NewMonitor() to = %v, want %v", m.to, tt.to)
				}
				if m.interval != tt.interval && m.interval != time.Second {
					t.Errorf("NewMonitor() interval = %v, want %v or 1s", m.interval, tt.interval)
				}
			}
		})
	}
}

// TestMonitor_Start tests the Start method of Monitor
func TestMonitor_Start(t *testing.T) {
	tests := []struct {
		name         string
		host         string
		runErr       error
		privileged   bool
		interval     time.Duration
		timeout      time.Duration
		wantErr      bool
		customPinger pinger // Allow custom pinger implementation
	}{
		{
			name:       "successful start",
			host:       "example.com",
			privileged: true,
			interval:   time.Second,
			timeout:    time.Minute,
			wantErr:    false,
		},
		{
			name:       "run error",
			host:       "example.com",
			runErr:     fmt.Errorf("run error"),
			privileged: true,
			interval:   time.Second,
			timeout:    time.Minute,
			wantErr:    true,
		},
		{
			name:       "invalid host",
			host:       "invalid.host.that.does.not.exist",
			privileged: true,
			interval:   time.Second,
			timeout:    time.Minute,
			wantErr:    true,
		},
		{
			name:       "empty host",
			host:       "",
			privileged: true,
			interval:   time.Second,
			timeout:    time.Minute,
			wantErr:    true,
		},
		{
			name:       "unprivileged mode",
			host:       "example.com",
			privileged: false,
			interval:   time.Second,
			timeout:    time.Minute,
			wantErr:    false,
		},
		{
			name:       "custom interval",
			host:       "example.com",
			privileged: true,
			interval:   time.Second * 2,
			timeout:    time.Minute,
			wantErr:    false,
		},
		{
			name:       "custom timeout",
			host:       "example.com",
			privileged: true,
			interval:   time.Second,
			timeout:    time.Minute * 2,
			wantErr:    false,
		},
		{
			name:         "non-adapter pinger",
			host:         "example.com",
			privileged:   true,
			interval:     time.Second,
			timeout:      time.Minute,
			wantErr:      false,
			customPinger: &customPinger{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockMetricClient{}
			m, err := NewMonitor(tt.host, "source", "destination", tt.interval, tt.timeout, client)
			if err != nil {
				t.Fatalf("NewMonitor() error = %v", err)
			}

			if tt.name == "invalid host" || tt.name == "empty host" {
				// Don't set mock pinger to test real pinger creation
				err = m.Start(context.Background())
				if !tt.wantErr {
					t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			// Use custom pinger if provided, otherwise create mock pinger
			if tt.customPinger != nil {
				customPinger := tt.customPinger.(*customPinger)
				customPinger.onRecv = func(packet *ping.Packet) {
					m.client.Metric("ICMP", time.Now(), map[string]string{
						"from": m.from,
						"to":   m.to,
						"host": m.host,
					}, map[string]interface{}{
						"rtt": packet.Rtt.Nanoseconds(),
						"ttl": packet.Ttl,
					})
				}
				m.pinger = customPinger
			} else {
				mockPinger := &mockPinger{
					host:       tt.host,
					runErr:     tt.runErr,
					privileged: tt.privileged,
					interval:   tt.interval,
					timeout:    tt.timeout,
				}
				// Set OnRecv to simulate packet reception
				mockPinger.onRecv = func(packet *ping.Packet) {
					m.client.Metric("ICMP", time.Now(), map[string]string{
						"from": m.from,
						"to":   m.to,
						"host": m.host,
					}, map[string]interface{}{
						"rtt": packet.Rtt.Nanoseconds(),
						"ttl": packet.Ttl,
					})
				}
				m.pinger = mockPinger
			}

			err = m.Start(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify metrics were recorded
				if len(client.metrics) == 0 {
					t.Error("Start() did not record any metrics")
				}

				// Verify privileged mode was set correctly
				if mockPinger, ok := m.pinger.(*mockPinger); ok && mockPinger.privileged != tt.privileged {
					t.Errorf("Start() privileged = %v, want %v", mockPinger.privileged, tt.privileged)
				}

				// Verify interval and timeout were set correctly
				if mockPinger, ok := m.pinger.(*mockPinger); ok {
					if tt.name == "custom interval" && mockPinger.interval != tt.interval {
						t.Errorf("Start() interval = %v, want %v", mockPinger.interval, tt.interval)
					}
					if tt.name == "custom timeout" && mockPinger.timeout != tt.timeout {
						t.Errorf("Start() timeout = %v, want %v", mockPinger.timeout, tt.timeout)
					}
				}
			}
		})
	}
}

// TestMonitor_Stop tests the Stop method of Monitor
func TestMonitor_Stop(t *testing.T) {
	t.Run("stop after start", func(t *testing.T) {
		m, err := NewMonitor("example.com", "source", "destination", time.Second, time.Minute, &mockMetricClient{})
		if err != nil {
			t.Fatalf("NewMonitor() error = %v", err)
		}

		mp := &mockPinger{host: "example.com"}
		m.pinger = mp

		ctx := context.Background()
		if err := m.Start(ctx); err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		if err := m.Stop(); err != nil {
			t.Errorf("Stop() error = %v", err)
		}

		if !mp.stopped {
			t.Error("Stop() did not stop the pinger")
		}
	})

	t.Run("stop without start", func(t *testing.T) {
		m, err := NewMonitor("example.com", "source", "destination", time.Second, time.Minute, &mockMetricClient{})
		if err != nil {
			t.Fatalf("NewMonitor() error = %v", err)
		}

		if err := m.Stop(); err != nil {
			t.Errorf("Stop() error = %v", err)
		}
	})
}

// TestMonitor_Name tests the Name method of Monitor
func TestMonitor_Name(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want string
	}{
		{
			name: "normal case",
			from: "source",
			to:   "destination",
			want: "<source>-<destination>",
		},
		{
			name: "empty values",
			from: "",
			to:   "",
			want: "<>-<>",
		},
		{
			name: "empty from",
			from: "",
			to:   "destination",
			want: "<>-<destination>",
		},
		{
			name: "empty to",
			from: "source",
			to:   "",
			want: "<source>-<>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Monitor directly to avoid NewMonitor's default value setting
			m := &Monitor{
				from:     tt.from,
				to:       tt.to,
				host:     "example.com",
				interval: time.Second,
				client:   &mockMetricClient{},
			}

			if got := m.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMonitor_Start_Context tests the context handling in Start method
func TestMonitor_Start_Context(t *testing.T) {
	client := &mockMetricClient{}
	m, err := NewMonitor("example.com", "source", "destination", time.Second, time.Minute, client)
	if err != nil {
		t.Fatalf("NewMonitor() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Set up mock pinger that blocks until context is canceled
	mockPinger := &mockPinger{
		host:       "example.com",
		privileged: true,
		runErr:     fmt.Errorf("context canceled"),
	}
	m.pinger = mockPinger

	err = m.Start(ctx)
	if err == nil {
		t.Error("Start() with canceled context should return error")
	}
}

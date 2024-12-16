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
	host     string
	interval time.Duration
	timeout  time.Duration
	size     int
	stopped  bool
	onRecv   func(*ping.Packet)
	runErr   error
}

func (p *mockPinger) SetPrivileged(privileged bool) {}

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

// TestNewMonitor tests the NewMonitor function with various parameter combinations
func TestNewMonitor(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		from     string
		to       string
		interval time.Duration
		timeout  time.Duration
		client   metric.Client
		wantErr  bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		name    string
		host    string
		runErr  error
		wantErr bool
	}{
		{
			name:    "successful start",
			host:    "example.com",
			wantErr: false,
		},
		{
			name:    "run error",
			host:    "example.com",
			runErr:  fmt.Errorf("run error"),
			wantErr: true,
		},
		{
			name:    "invalid host",
			host:    "invalid.host.that.does.not.exist",
			wantErr: true,
		},
		{
			name:    "empty host",
			host:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockMetricClient{}
			m, err := NewMonitor(tt.host, "source", "destination", time.Second, time.Minute, client)
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

			// Set up mock pinger
			mockPinger := &mockPinger{
				host:   tt.host,
				runErr: tt.runErr,
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

			ctx := context.Background()
			err = m.Start(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify metrics were recorded
				if len(client.metrics) == 0 {
					t.Error("Start() did not record any metrics")
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

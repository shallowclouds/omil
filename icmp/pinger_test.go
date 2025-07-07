package icmp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/shallowclouds/omil/metric"
	"github.com/stretchr/testify/require"
)

type mockMetricClient struct {
	metrics []MetricCall
}

type MetricCall struct {
	Name      string
	Timestamp time.Time
	Tags      map[string]string
	Value     map[string]interface{}
}

func (m *mockMetricClient) Metric(name string, timestamp time.Time, tags map[string]string, value map[string]interface{}) {
	m.metrics = append(m.metrics, MetricCall{
		Name:      name,
		Timestamp: timestamp,
		Tags:      tags,
		Value:     value,
	})
}

func (m *mockMetricClient) Flush() error { return nil }
func (m *mockMetricClient) Exit() error  { return nil }

var _ metric.Client = (*mockMetricClient)(nil)

func TestNewMonitor(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		from     string
		to       string
		interval time.Duration
		timeout  time.Duration
		client   metric.Client
		hasError bool
	}{
		{
			name:     "valid monitor creation",
			host:     "127.0.0.1",
			from:     "localhost",
			to:       "target",
			interval: time.Second,
			timeout:  time.Minute,
			client:   &mockMetricClient{},
			hasError: false,
		},
		{
			name:     "nil client should error",
			host:     "127.0.0.1",
			from:     "localhost",
			to:       "target",
			interval: time.Second,
			timeout:  time.Minute,
			client:   nil,
			hasError: true,
		},
		{
			name:     "empty from defaults to hostname",
			host:     "8.8.8.8",
			from:     "",
			to:       "google-dns",
			interval: time.Second * 2,
			timeout:  time.Minute * 2,
			client:   &mockMetricClient{},
			hasError: false,
		},
		{
			name:     "empty to defaults to host",
			host:     "1.1.1.1",
			from:     "test-host",
			to:       "",
			interval: time.Second * 3,
			timeout:  time.Minute * 3,
			client:   &mockMetricClient{},
			hasError: false,
		},
		{
			name:     "zero interval defaults to 1 second",
			host:     "127.0.0.1",
			from:     "localhost",
			to:       "target",
			interval: 0,
			timeout:  time.Minute,
			client:   &mockMetricClient{},
			hasError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			monitor, err := NewMonitor(testCase.host, testCase.from, testCase.to, testCase.interval, testCase.timeout, testCase.client)

			if testCase.hasError {
				require.Error(t, err)
				require.Nil(t, monitor)
			} else {
				require.NoError(t, err)
				require.NotNil(t, monitor)
				require.Equal(t, testCase.host, monitor.host)

				if testCase.from == "" {
					hostname, _ := os.Hostname()
					if hostname == "" {
						hostname = "localhost"
					}
					require.Equal(t, hostname, monitor.from)
				} else {
					require.Equal(t, testCase.from, monitor.from)
				}

				if testCase.to == "" {
					require.Equal(t, testCase.host, monitor.to)
				} else {
					require.Equal(t, testCase.to, monitor.to)
				}

				expectedInterval := testCase.interval
				if expectedInterval == 0 {
					expectedInterval = time.Second
				}
				require.Equal(t, expectedInterval, monitor.interval)
				require.Equal(t, testCase.timeout, monitor.timeout)
			}
		})
	}
}

func TestMonitor_Name(t *testing.T) {
	testCases := []struct {
		name     string
		from     string
		to       string
		expected string
	}{
		{
			name:     "basic name format",
			from:     "from",
			to:       "to",
			expected: "<from>-<to>",
		},
		{
			name:     "with special characters",
			from:     "test-host",
			to:       "google-dns",
			expected: "<test-host>-<google-dns>",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &mockMetricClient{}
			monitor, err := NewMonitor("127.0.0.1", testCase.from, testCase.to, time.Second, time.Minute, client)
			require.NoError(t, err)

			require.Equal(t, testCase.expected, monitor.Name())
		})
	}
}

func TestMonitor_Stop(t *testing.T) {
	testCases := []struct {
		name        string
		setupPinger bool
		expectError bool
	}{
		{
			name:        "stop with nil pinger",
			setupPinger: false,
			expectError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := &mockMetricClient{}
			monitor, err := NewMonitor("127.0.0.1", "from", "to", time.Second, time.Minute, client)
			require.NoError(t, err)

			err = monitor.Stop()
			if testCase.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMonitor_Start_ContextCancellation(t *testing.T) {
	client := &mockMetricClient{}
	monitor, err := NewMonitor("127.0.0.1", "from", "to", time.Millisecond*100, time.Second, client)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
	defer cancel()

	err = monitor.Start(ctx)
	require.Error(t, err)
	require.True(t,
		err.Error() == "context deadline exceeded" ||
			err.Error() == "context canceled" ||
			ctx.Err() != nil ||
			err.Error() == "failed to run pinger: listen ip4:icmp : socket: operation not permitted" ||
			err.Error() == "failed to create pinger: listen ip4:icmp : socket: operation not permitted",
		"Expected context cancellation or permission error, got: %v", err)
}

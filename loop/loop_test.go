package loop

import (
	"context"
	"testing"
	"time"

	"github.com/shallowclouds/omil/icmp"
	"github.com/shallowclouds/omil/metric"
	"github.com/stretchr/testify/require"
)

type mockMonitor struct {
	name      string
	startErr  error
	stopErr   error
	startChan chan struct{}
}

func (m *mockMonitor) Start(ctx context.Context) error {
	if m.startChan != nil {
		select {
		case <-m.startChan:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return m.startErr
}

func (m *mockMonitor) Stop() error {
	return m.stopErr
}

func (m *mockMonitor) Name() string {
	return m.name
}

type mockMetricClient struct{}

func (m *mockMetricClient) Metric(name string, timestamp time.Time, tags map[string]string, value map[string]interface{}) {
}
func (m *mockMetricClient) Flush() error { return nil }
func (m *mockMetricClient) Exit() error  { return nil }

var _ metric.Client = (*mockMetricClient)(nil)

func TestLoop(t *testing.T) {
	testCases := []struct {
		name     string
		monitors []*icmp.Monitor
		timeout  time.Duration
		hasError bool
	}{
		{
			name:     "empty monitors",
			monitors: []*icmp.Monitor{},
			timeout:  time.Millisecond * 100,
			hasError: true,
		},
		{
			name: "single monitor with context timeout",
			monitors: func() []*icmp.Monitor {
				client := &mockMetricClient{}
				monitor, _ := icmp.NewMonitor("127.0.0.1", "from", "to", time.Millisecond*10, time.Second, client)
				return []*icmp.Monitor{monitor}
			}(),
			timeout:  time.Millisecond * 50,
			hasError: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), testCase.timeout)
			defer cancel()

			err := Loop(ctx, testCase.monitors)

			if testCase.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestErrInterrupt(t *testing.T) {
	require.NotNil(t, ErrInterrupt)
	require.Equal(t, "signal interrupt", ErrInterrupt.Error())
}

package metric

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockClient struct {
	metrics  []MetricCall
	flushErr error
	exitErr  error
}

type MetricCall struct {
	Name      string
	Timestamp time.Time
	Tags      map[string]string
	Value     map[string]interface{}
}

func (m *mockClient) Metric(name string, timestamp time.Time, tags map[string]string, value map[string]interface{}) {
	m.metrics = append(m.metrics, MetricCall{
		Name:      name,
		Timestamp: timestamp,
		Tags:      tags,
		Value:     value,
	})
}

func (m *mockClient) Flush() error {
	return m.flushErr
}

func (m *mockClient) Exit() error {
	return m.exitErr
}

func TestClientInterface(t *testing.T) {
	client := &mockClient{}

	var _ Client = client

	now := time.Now()
	tags := map[string]string{"host": "test"}
	value := map[string]interface{}{"rtt": 100}

	client.Metric("ICMP", now, tags, value)

	require.Len(t, client.metrics, 1)
	require.Equal(t, "ICMP", client.metrics[0].Name)
	require.Equal(t, now, client.metrics[0].Timestamp)
	require.Equal(t, tags, client.metrics[0].Tags)
	require.Equal(t, value, client.metrics[0].Value)

	require.NoError(t, client.Flush())
	require.NoError(t, client.Exit())
}

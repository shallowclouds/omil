package influxdb

import (
	"fmt"
	"testing"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// mockClient mocks the influxDB2 client
type mockClient struct {
	influxdb2.Client
}

func (c *mockClient) Close() {}

// mockWriteAPI mocks the influxDB2 write api
type mockWriteAPI struct {
	api.WriteAPI
	errChan <-chan error
}

func (a *mockWriteAPI) Flush() {}

func (a *mockWriteAPI) Errors() <-chan error {
	return a.errChan
}

func TestClientV2_Stop(t *testing.T) {
	const errNum = 10
	errChan := make(chan error)
	cli := &ClientV2{
		cli: &mockClient{},
		api: &mockWriteAPI{
			errChan: errChan,
		},
		stopChan: make(chan struct{}),
	}
	go func() {
		for i := 0; i < errNum; i++ {
			errChan <- fmt.Errorf("sample error %d", i)
		}
		_ = cli.Exit()
	}()
	cli.logError()
	close(errChan)
}

package influxdb

import (
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/sirupsen/logrus"
)

// ClientV2 sends data points to InfluxDB v2.* server.
type ClientV2 struct {
	cli      influxdb2.Client
	api      api.WriteAPI
	stopChan chan struct{}
}

// NewClientV2 constructs a InfluxDB v2.* client.
func NewClientV2(addr, org, bucket, token string) (*ClientV2, error) {
	cli := influxdb2.NewClientWithOptions(addr, token, influxdb2.DefaultOptions().SetBatchSize(10))
	writeAPI := cli.WriteAPI(org, bucket)
	stopChan := make(chan struct{})
	client := &ClientV2{
		cli:      cli,
		api:      writeAPI,
		stopChan: stopChan,
	}
	// start error log loop
	go client.logError()
	return client, nil
}

// Metric emits a data point to InfluxDB server asynchronously.
func (c *ClientV2) Metric(name string, timestamp time.Time, tags map[string]string, value map[string]interface{}) {
	p := influxdb2.NewPoint(
		name,
		tags,
		value,
		timestamp,
	)
	c.api.WritePoint(p)
}

// Flush waits the influx DB client sending all data points to server.
func (c *ClientV2) Flush() error {
	c.api.Flush()
	return nil
}

// Exit wait for client flushing and closing.
func (c *ClientV2) Exit() error {
	_ = c.Flush()
	c.cli.Close()
	close(c.stopChan)
	return nil
}

// logError loops on writeAPI.Errors and log errors out.
func (c *ClientV2) logError() {
	for {
		select {
		case err := <-c.api.Errors():
			if err == nil {
				logrus.Info("InfluxDB2 error channel closed, exited")
				return
			}
			logrus.WithError(err).Warn("failed to send data point to server")
		case <-c.stopChan:
			logrus.Info("InfluxDB2 error log exited")
			return
		}
	}
}

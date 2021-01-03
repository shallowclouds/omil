package influxdb

import (
	"sync"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/sirupsen/logrus"
)

// Deprecated: Client sends data points to InfluxDB v1.* server asynchronously.
type Client struct {
	influxCli client.Client
	wg        sync.WaitGroup
	pointChan chan *client.Point
	doneChan  chan struct{}
	db        string
}

const (
	DefaultBufferSize    = 10240
	DefaultBatchSize     = 10
	DefaultConsumerCount = 1
)

// Deprecated: NewAsyncClient creates an async client to send data points to InfluxDB server.
func NewAsyncClient(addr, db, username, password string) (*Client, error) {
	cli, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     addr,
		Username: username,
		Password: password,
	})
	if err != nil {
		return nil, err
	}
	pointChan := make(chan *client.Point, DefaultBufferSize)
	doneChan := make(chan struct{})

	c := &Client{
		influxCli: cli,
		wg:        sync.WaitGroup{},
		pointChan: pointChan,
		doneChan:  doneChan,
		db:        db,
	}

	for i := 0; i < DefaultConsumerCount; i++ {
		go c.consume()
		c.wg.Add(1)
	}

	return c, nil
}

func (c *Client) Metric(name string, timestamp time.Time, tags map[string]string, value map[string]interface{}) {
	point, err := client.NewPoint(name, tags, value, timestamp)
	if err != nil {
		logrus.WithError(err).Warn("failed to create point")
		return
	}
	c.pointChan <- point
}

func (c *Client) consume() {
	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: c.db,
	})
	if err != nil {
		logrus.WithError(err).Panic("failed to create batch points")
	}
	size := 0
	for {
		var point *client.Point
		select {
		case point = <-c.pointChan:
		case <-c.doneChan:
			goto end
		}
		batch.AddPoint(point)
		size++
		if size < DefaultBatchSize {
			continue
		}
		if err := c.influxCli.Write(batch); err != nil {
			logrus.WithError(err).Error("failed to write points to influx db")
			continue
		}
		batch, err = client.NewBatchPoints(client.BatchPointsConfig{
			Database: c.db,
		})
		if err != nil {
			logrus.WithError(err).Panic("failed to create batch points")
		}
		size = 0
	}
end:
	if err := c.influxCli.Write(batch); err != nil {
		logrus.WithError(err).Error("failed to write points to influx db")
	}
	c.wg.Done()
}

func (c *Client) Flush() error {
	c.doneChan <- struct{}{}
	c.wg.Wait()
	return nil
}

func (c *Client) Stop() error {
	_ = c.Flush()
	return c.influxCli.Close()
}

package icmp

import (
	"context"
	"fmt"
	"github.com/go-ping/ping"
	"github.com/pkg/errors"
	"github.com/shallowclouds/omil/metric"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

type Monitor struct {
	from, to string
	host     string
	interval time.Duration
	client   metric.Client
	pinger   *ping.Pinger
}

func NewMonitor(host, from, to string, interval time.Duration, client metric.Client) (*Monitor, error) {
	var err error
	if from == "" {
		from, err = os.Hostname()
		if err != nil {
			logrus.WithError(err).Warn("get hostname err")
			from = "localhost"
		}
	}
	if to == "" {
		to = host
	}

	if interval == 0 {
		interval = time.Second
	}

	if client == nil {
		return nil, fmt.Errorf("nil metric client")
	}

	return &Monitor{
		from:     from,
		to:       to,
		host:     host,
		interval: interval,
		client:   client,
	}, nil
}

func (m *Monitor) Start(ctx context.Context) error {
	pinger, err := ping.NewPinger(m.host)
	if err != nil {
		return errors.Wrap(err, "failed to create pinger")
	}

	pinger.Interval = m.interval
	// true for ICMP
	pinger.SetPrivileged(true)
	pinger.Size = 64
	pinger.OnRecv = func(packet *ping.Packet) {
		logrus.WithFields(logrus.Fields{
			"rtt": packet.Rtt,
			"nbytes": packet.Nbytes,
			"seq": packet.Seq,
			"ttl": packet.Ttl,
		}).Info("recv ICMP packet")
		m.client.Metric("ICMP", map[string]string{
			"from": m.from,
			"to": m.to,
			"host": m.host,
		}, map[string]interface{}{
			"rtt": packet.Rtt.Nanoseconds(),
			"ttl": packet.Ttl,
		})
	}

	m.pinger = pinger

	if err := pinger.Run(); err != nil {
		return errors.Wrap(err, "failed to run pinger")
	}

	return nil
}

func (m *Monitor) Stop() error {
	if m.pinger == nil {
		return nil
	}
	m.pinger.Stop()
	return nil
}

func (m *Monitor) Name() string {
	return fmt.Sprintf("<%s>-<%s>", m.from, m.to)
}

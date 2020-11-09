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
	timeout  time.Duration
}

func NewMonitor(host, from, to string, interval, timeout time.Duration, client metric.Client) (*Monitor, error) {
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
		timeout:  timeout,
	}, nil
}

func (m *Monitor) Start(ctx context.Context) error {
	pinger, err := ping.NewPinger(m.host)
	if err != nil {
		return errors.Wrap(err, "failed to create pinger")
	}

	if m.timeout != 0 {
		pinger.Timeout = m.timeout
	}
	pinger.Interval = m.interval
	// true for ICMP
	pinger.SetPrivileged(true)
	pinger.Size = 64

	// use startTime to calculate packet sent time: startTime + interval * packet_sequence_number
	// TODO: use a more accurate and graceful way
	var startTime time.Time

	pinger.OnRecv = func(packet *ping.Packet) {
		logrus.WithFields(logrus.Fields{
			"rtt":    packet.Rtt,
			"nbytes": packet.Nbytes,
			"seq":    packet.Seq,
			"ttl":    packet.Ttl,
		}).Info("recv ICMP packet")
		m.client.Metric("ICMP", startTime.Add(time.Duration(packet.Seq)*m.interval), map[string]string{
			"from": m.from,
			"to":   m.to,
			"host": m.host,
		}, map[string]interface{}{
			"rtt": packet.Rtt.Nanoseconds(),
			"ttl": packet.Ttl,
		})
	}

	m.pinger = pinger

	startTime = time.Now()
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

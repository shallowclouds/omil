package icmp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-ping/ping"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/shallowclouds/omil/metric"
)

// Monitor sends and receives ICMP packet to the specified host.
type Monitor struct {
	from, to string
	host     string
	interval time.Duration
	client   metric.Client
	pinger   *ping.Pinger
	timeout  time.Duration
}

// NewMonitor creates a ICMP network monitor.
// `host` is the target hostname need to test.
// `from` is the name of this server, used as metric tag `from`.
// `to` is the name of target host, used as metric tag `to`.
// `interval` specifies the time interval to send ICMP packets.
// `timeout` specifies the time to end the loop, 0 for infinite.
// `client` is the metric client to send data points.
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
		return nil, errors.New("nil metric client")
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

// Start starts the loop to test network and send data points.
//
// Return an error if loop failed.
func (m *Monitor) Start(_ context.Context) error {
	pinger, err := ping.NewPinger(m.host)
	if err != nil {
		return errors.WithMessage(err, "failed to create pinger")
	}

	if m.timeout != 0 {
		pinger.Timeout = m.timeout
	}
	pinger.Interval = m.interval
	// True for ICMP
	pinger.SetPrivileged(true)
	pinger.Size = 64

	// Use startTime to calculate packet sent time: startTime + interval * packet_sequence_number,
	// as will cant put the send time in the ICMP packet data at present.
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

	sendTicker := time.NewTicker(m.interval)
	defer sendTicker.Stop()

	startTime = time.Now()
	go func() {
		// Metrics for sending packets.
		for t := range sendTicker.C {
			if t.IsZero() {
				break
			}
			m.client.Metric("ICMP", t, map[string]string{
				"from": m.from,
				"to":   m.to,
				"host": m.host,
			}, map[string]interface{}{
				"sent": 1,
			})
		}
	}()
	if err := pinger.Run(); err != nil {
		return errors.WithMessage(err, "failed to run pinger")
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

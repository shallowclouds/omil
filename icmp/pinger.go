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

// For testing purposes
var hostnameFunc = os.Hostname

// pinger defines the interface for ICMP ping operations
type pinger interface {
	SetPrivileged(privileged bool)
	Run() error
	Stop()
}

// pingAdapter adapts ping.Pinger to our pinger interface
type pingAdapter struct {
	*ping.Pinger
}

// newPinger creates a new pinger for the given host
func newPinger(host string) (pinger, error) {
	p, err := ping.NewPinger(host)
	if err != nil {
		return nil, err
	}
	return &pingAdapter{Pinger: p}, nil
}

// Monitor sends and receives ICMP packet to the specified host.
type Monitor struct {
	from, to string
	host     string
	interval time.Duration
	client   metric.Client
	pinger   pinger
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
	// Only set default hostname if from is empty and host is not empty
	if from == "" && host != "" {
		from, err = hostnameFunc()
		if err != nil {
			return nil, errors.WithMessage(err, "failed to get hostname")
		}
	}
	// Only set default to if it's empty and host is not empty
	if to == "" && host != "" {
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
	if m.pinger == nil {
		p, err := newPinger(m.host)
		if err != nil {
			return errors.WithMessage(err, "failed to create pinger")
		}
		m.pinger = p
	}

	// Use startTime to calculate packet sent time: startTime + interval * packet_sequence_number,
	// as will cant put the send time in the ICMP packet data at present.
	// TODO: use a more accurate and graceful way
	startTime := time.Now()

	if adapter, ok := m.pinger.(*pingAdapter); ok {
		if m.timeout != 0 {
			adapter.Timeout = m.timeout
		}
		adapter.Interval = m.interval
		adapter.Size = 64

		adapter.OnRecv = func(packet *ping.Packet) {
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
	}

	// Set privileged mode based on the pinger type
	if adapter, ok := m.pinger.(*pingAdapter); ok {
		adapter.SetPrivileged(false) // Default to unprivileged mode for security
	}

	sendTicker := time.NewTicker(m.interval)
	defer sendTicker.Stop()

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

	if err := m.pinger.Run(); err != nil {
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

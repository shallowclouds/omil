package loop

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/shallowclouds/omil/icmp"
)

var (
	ErrInterrupt = errors.New("signal interrupt")

	restartInterval = time.Second
)

func Loop(ctx context.Context, monitors []*icmp.Monitor) (err error) {
	if len(monitors) == 0 {
		return errors.New("no monitors provided")
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	go func() {
		select {
		case sig := <-sigChan:
			err = ErrInterrupt
			logrus.Infof("Recv signal %s, exiting...", sig.String())
			cancel()
		case <-ctx.Done():
		}
	}()

	for _, monitor := range monitors {
		wg.Add(1)
		go func(m *icmp.Monitor) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					logrus.Infof("exiting monitor %s", m.Name())
					if err := m.Stop(); err != nil {
						logrus.WithError(err).Error("failed to stop monitor")
					}
					return
				default:
					monitorCtx, monitorCancel := context.WithTimeout(ctx, time.Minute*5)

					if err := m.Start(monitorCtx); err != nil {
						if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
							monitorCancel()
							logrus.Infof("monitor %s cancelled or timed out", m.Name())
							return
						}
						logrus.WithError(err).Errorf("monitor %s failed, restarting in %v", m.Name(), restartInterval)
					}

					monitorCancel()

					select {
					case <-ctx.Done():
						return
					case <-time.After(restartInterval):
						logrus.Infof("restarting monitor %s", m.Name())
					}
				}
			}
		}(monitor)
	}

	wg.Wait()
	return err
}

package loop

import (
	"context"
	"math"
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

	baseRestartInterval = time.Second
	maxRestartInterval  = time.Minute * 5
	maxRetries          = 3
)

func Loop(ctx context.Context, monitors []*icmp.Monitor) (err error) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		sig := <-sigChan
		err = ErrInterrupt
		logrus.Infof("Recv signal %s, exiting...", sig.String())
		cancel()
	}()

	restart := true
	var mu sync.RWMutex
	go func() {
		<-ctx.Done()
		mu.Lock()
		restart = false
		mu.Unlock()
		for _, monitor := range monitors {
			logrus.Infof("stopping monitor %s", monitor.Name())
			if err := monitor.Stop(); err != nil {
				logrus.WithError(err).Error("failed to stop monitor")
			}
		}
	}()
	for _, monitor := range monitors {
		wg.Add(1)
		m := monitor
		go func() {
			retryCount := 0
			currentInterval := baseRestartInterval

			for {
				mu.RLock()
				if !restart {
					logrus.Infof("exiting monitor %s", m.Name())
					wg.Done()
					break
				}
				mu.RUnlock()

				if err := m.Start(ctx); err != nil {
					retryCount++
					wrappedErr := errors.Wrapf(err, "failed to run monitor %s (attempt %d/%d)", m.Name(), retryCount, maxRetries)
					logrus.WithError(wrappedErr).Error("monitor failed")

					if retryCount >= maxRetries {
						logrus.WithError(wrappedErr).Errorf("monitor %s exceeded max retries, using exponential backoff", m.Name())
						backoffMultiplier := math.Pow(2, float64(retryCount-maxRetries))
						currentInterval = time.Duration(float64(baseRestartInterval) * backoffMultiplier)
						if currentInterval > maxRestartInterval {
							currentInterval = maxRestartInterval
						}
					}
				} else {
					retryCount = 0
					currentInterval = baseRestartInterval
				}

				time.Sleep(currentInterval)

				mu.RLock()
				if restart {
					if retryCount > 0 {
						logrus.Infof("retrying monitor %s (attempt %d)", m.Name(), retryCount+1)
					} else {
						logrus.Infof("restarting monitor %s", m.Name())
					}
				}
				mu.RUnlock()
			}
		}()
	}
	wg.Wait()
	return nil
}

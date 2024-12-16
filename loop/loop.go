package loop

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Monitor represents a monitor that can be started and stopped
type Monitor interface {
	Start(ctx context.Context) error
	Stop() error
	Name() string
}

var (
	ErrInterrupt = errors.New("signal interrupt")

	restartInterval = time.Second
)

func Loop(ctx context.Context, monitors []Monitor) (err error) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	errChan := make(chan error, 1)
	go func() {
		sig := <-sigChan
		errChan <- ErrInterrupt
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
			defer wg.Done()
			for {
				mu.RLock()
				if !restart {
					logrus.Infof("exiting monitor %s", m.Name())
					mu.RUnlock()
					return
				}
				mu.RUnlock()

				if err := m.Start(ctx); err != nil {
					logrus.WithError(err).Error("failed to run monitor")
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(restartInterval):
					mu.RLock()
					if restart {
						logrus.Infof("restarting monitor %s", m.Name())
					}
					mu.RUnlock()
				}
			}
		}()
	}

	wg.Wait()
	select {
	case err = <-errChan:
		return err
	default:
		return nil
	}
}

package loop

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/shallowclouds/omil/icmp"
)

var (
	ErrInterrupt = errors.New("signal interrupt")

	restartInterval = time.Second
)

func Loop(ctx context.Context, monitors []*icmp.Monitor) (err error) {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigChan := make(chan os.Signal)
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
			for {
				mu.RLock()
				if !restart {
					logrus.Infof("exiting monitor %s", m.Name())
					wg.Done()
					break
				}
				mu.RUnlock()
				if err := m.Start(ctx); err != nil {
					logrus.WithError(err).Error("failed to run monitor")
				}
				time.Sleep(restartInterval)

				mu.RLock()
				if restart {
					logrus.Infof("restarting monitor %s", m.Name())
				}
				mu.RUnlock()
			}
		}()
	}
	wg.Wait()
	return nil
}

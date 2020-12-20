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

	for _, monitor := range monitors {
		wg.Add(1)
		monitor := monitor
		go func() {
			restart := true
			var mu sync.RWMutex
			go func() {
				<-ctx.Done()
				logrus.Infof("stopping monitor %s", monitor.Name())
				if err := monitor.Stop(); err != nil {
					logrus.WithError(err).Error("failed to stop monitor")
				}
				mu.Lock()
				restart = false
				mu.Unlock()
			}()
			for {
				if err := monitor.Start(ctx); err != nil {
					logrus.WithError(err).Error("failed to run monitor")
				}
				mu.RLock()
				if !restart {
					logrus.Infof("exiting monitor %s", monitor.Name())
					wg.Done()
					break
				}
				mu.RUnlock()
				time.Sleep(restartInterval)
				logrus.Infof("restarting monitor %s", monitor.Name())
			}
		}()
	}
	wg.Wait()
	return nil
}

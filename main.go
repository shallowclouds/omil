package main

import (
	"context"
	"fmt"
	"github.com/shallowclouds/omil/config"
	"github.com/shallowclouds/omil/icmp"
	"github.com/shallowclouds/omil/influxdb"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	compiledTimeString string
	version            string
)

func init() {

}

func main() {
	app := cli.App{
		Name:        "Omil",
		HelpName:    "help",
		Usage:       "omil --config <config_file_path>",
		ArgsUsage:   "",
		Version:     fmt.Sprintf("\ngit version: %s\nbuild time: %s", version, compiledTimeString),
		Description: fmt.Sprintf("Simple tool for monitoring network latency, build %s", version),
		Commands:    nil,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "--config /path/to/config/file",
				EnvVars: []string{
					"CONFIG_FILE",
				},
				Required: false,
				Value:    "conf/config.yaml",
			},
		},
		EnableBashCompletion: false,
		HideHelp:             false,
		HideHelpCommand:      false,
		HideVersion:          false,
		BashComplete:         nil,
		Before:               nil,
		After:                nil,
		Action: func(ctx *cli.Context) error {
			configFile := ctx.String("config")
			if configFile != "" {
				config.SetConfigFilePath(configFile)
			}
			conf := config.Config()
			metricClient, err := influxdb.NewAsyncClient(conf.InfluxDBv1.Addr, conf.InfluxDBv1.Database, conf.InfluxDBv1.Username, conf.InfluxDBv1.Password)
			if err != nil {
				logrus.WithError(err).Fatal("failed to create influx db client")
			}

			monitors := make([]*icmp.Monitor, 0, len(conf.Targets))
			for _, t := range conf.Targets {
				monitor, err := icmp.NewMonitor(t.Host, conf.Hostname, t.Name, time.Second, metricClient)
				if err != nil {
					logrus.WithError(err).WithFields(logrus.Fields{
						"target_host": t.Host,
						"target_name": t.Name,
					}).Error("failed to create monitor, skipped")
					continue
				}
				monitors = append(monitors, monitor)
			}

			var wg sync.WaitGroup
			c, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal)
			signal.Notify(sigChan, os.Interrupt)

			go func() {
				sig := <-sigChan
				logrus.Infof("Recv signal %s, exiting...", sig.String())
				cancel()
			}()

			for _, monitor := range monitors {
				wg.Add(1)
				monitor := monitor
				go func() {
					go func() {
						<-c.Done()
						logrus.Infof("stopping monitor %s", monitor.Name())
						if err := monitor.Stop(); err != nil {
							logrus.WithError(err).Error("failed to stop monitor")
						}
					}()
					if err := monitor.Start(c); err != nil {
						logrus.WithError(err).Error("failed to run monitor")
					}

					wg.Done()
				}()
			}
			wg.Wait()
			logrus.Info("bye~")
			return nil
		},
		CommandNotFound: nil,
		OnUsageError:    nil,
		Authors: []*cli.Author{
			{
				Name:  "Yorling",
				Email: "ishallowcloud@gmail.com",
			},
		},
		Copyright:              "",
		UseShortOptionHandling: true,
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Fatal("failed to run commands")
	}
}

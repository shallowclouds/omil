package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/shallowclouds/omil/config"
	"github.com/shallowclouds/omil/icmp"
	"github.com/shallowclouds/omil/influxdb"
	"github.com/shallowclouds/omil/loop"
	"github.com/shallowclouds/omil/metric"
)

var (
	compiledTimeString string
	version            string
)

type App struct {
	config       *config.ConfigStruct
	metricClient metric.Client
}

func NewApp(configFile string) (*App, error) {
	if configFile != "" {
		config.SetConfigFilePath(configFile)
	}

	conf := config.Config()
	if err := conf.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid configuration")
	}

	metricClient, err := influxdb.NewClientV2(conf.InfluxDBv2.Addr, conf.InfluxDBv2.Org, conf.InfluxDBv2.Bucket, conf.InfluxDBv2.Token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create influx db client")
	}

	return &App{
		config:       conf,
		metricClient: metricClient,
	}, nil
}

func (app *App) CreateMonitors() ([]*icmp.Monitor, error) {
	targets := app.config.GetTargets()
	hostname := app.config.GetHostname()

	monitors := make([]*icmp.Monitor, 0, len(targets))
	for _, t := range targets {
		// As pinger stores all packets data in memory,
		// so too long timeout may cause high memory usage.
		// Just let it stop and restart.
		monitor, err := icmp.NewMonitor(t.Host, hostname, t.Name, time.Second, time.Minute*60, app.metricClient)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"target_host": t.Host,
				"target_name": t.Name,
			}).Error("failed to create monitor, skipped")
			continue
		}
		monitors = append(monitors, monitor)
	}

	if len(monitors) == 0 {
		return nil, errors.New("no valid monitors created")
	}

	return monitors, nil
}

func (app *App) Run(ctx context.Context) error {
	monitors, err := app.CreateMonitors()
	if err != nil {
		return errors.Wrap(err, "failed to create monitors")
	}

	if err := loop.Loop(ctx, monitors); err != nil {
		if errors.Is(err, loop.ErrInterrupt) {
			logrus.WithError(err).Info("monitor loop exited gracefully")
			return nil
		}
		return errors.Wrap(err, "monitor loop failed")
	}

	logrus.Info("bye~")
	return nil
}

func mainAction(ctx *cli.Context) error {
	app, err := NewApp(ctx.String("config"))
	if err != nil {
		return err
	}

	return app.Run(ctx.Context)
}

func main() {
	app := cli.App{
		Name:        "Omil",
		HelpName:    "help",
		Usage:       "omil --config <config_file_path>",
		ArgsUsage:   "",
		Version:     fmt.Sprintf("\ngit version: %s\nbuild time: %s", version, compiledTimeString),
		Description: fmt.Sprintf("Simple tool for monitoring network latency, build %s", version),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "--config /path/to/config/file",
				EnvVars: []string{
					"CONFIG_FILE",
				},
				Value: "conf/config.yaml",
			},
		},
		Action: mainAction,
		Authors: []*cli.Author{
			{
				Name:  "Yorling",
				Email: "ishallowcloud@gmail.com",
			},
		},
		UseShortOptionHandling: true,
	}

	if err := app.Run(os.Args); err != nil {
		logrus.WithError(err).Fatal("failed to run commands")
	}
}

package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/shallowclouds/omil/config"
	"github.com/shallowclouds/omil/icmp"
	"github.com/shallowclouds/omil/influxdb"
	"github.com/shallowclouds/omil/loop"
)

var (
	compiledTimeString string
	version            string
)

func mainAction(ctx *cli.Context) error {
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
		monitor, err := icmp.NewMonitor(t.Host, conf.Hostname, t.Name, time.Second, time.Hour*12, metricClient)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"target_host": t.Host,
				"target_name": t.Name,
			}).Error("failed to create monitor, skipped")
			continue
		}
		monitors = append(monitors, monitor)
	}

	if err := loop.Loop(ctx.Context, monitors); err != nil {
		if errors.Is(err, loop.ErrInterrupt) {
			logrus.WithError(err).Error("monitor loop exited")
			return nil
		}
		logrus.WithError(err).Error("monitor loop broke")
		return err
	}

	logrus.Info("bye~")
	return nil
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

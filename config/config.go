package config

import (
	"os"
	"sync"

	"github.com/jinzhu/configor"
	"github.com/sirupsen/logrus"
)

type Target struct {
	Host string `yaml:"Host"`
	Name string `yaml:"Name"`
}

type configStruct struct {
	Hostname string `yaml:"Hostname"`
	// Deprecated: InfluxDB 1.* sever configs.
	InfluxDBv1 struct {
		Addr     string `yaml:"Addr"`
		Username string `yaml:"Username"`
		Password string `yaml:"Password"`
		Database string `yaml:"Database"`
	} `yaml:"InfluxDBv1"`
	// InfluxDB 2.* sever configs.
	InfluxDBv2 struct {
		Addr   string `yaml:"Addr"`
		Org    string `yaml:"Org"`
		Bucket string `yaml:"Bucket"`
		Token  string `yaml:"Token"`
	} `yaml:"InfluxDBv2"`
	Targets []Target `yaml:"Targets"`
}

var (
	configFilePath string
	initConfigOnce sync.Once
	config         *configStruct
	// Override os.Exit for testing
	osExit = os.Exit
)

func SetConfigFilePath(filepath string) {
	configFilePath = filepath
}

func Config() *configStruct {
	initConfigOnce.Do(func() {
		config = new(configStruct)
		paths := []string{configFilePath}
		if configFilePath == "" {
			paths = []string{"conf/config.yml"}
		}

		// Check if the config file exists and is accessible
		configPath := paths[0]
		if _, err := os.Stat(configPath); err != nil {
			// Any error accessing the file should be treated as file not found
			logrus.WithField("path", configPath).WithError(err).Fatal("config file not found")
			osExit(1)
			return
		}

		// Try to load config from specified paths
		if err := configor.Load(config, paths...); err != nil {
			logrus.WithError(err).Fatal("failed to load config from file")
			osExit(1)
			return
		}

		// Only validate fields if we successfully loaded the config
		if config.Hostname == "" {
			logrus.Fatal("hostname is required in config")
			osExit(1)
			return
		}
	})
	return config
}

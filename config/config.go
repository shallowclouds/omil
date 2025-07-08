package config

import (
	"sync"

	"github.com/jinzhu/configor"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Target struct {
	Host string `yaml:"Host"`
	Name string `yaml:"Name"`
}

type configStruct struct {
	Hostname       string `yaml:"Hostname"`
	TimeoutMinutes int    `yaml:"TimeoutMinutes"`
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
)

func SetConfigFilePath(filepath string) {
	configFilePath = filepath
}

func Config() *configStruct {
	initConfigOnce.Do(func() {
		config = new(configStruct)
		if err := configor.Load(config, configFilePath, "conf/config.yml"); err != nil {
			logrus.WithError(err).Fatal("failed to load config from file")
		}

		if config.TimeoutMinutes <= 0 {
			config.TimeoutMinutes = 60
		}

		if err := validateConfig(config); err != nil {
			logrus.WithError(err).Fatal("configuration validation failed")
		}
	})
	return config
}

func validateConfig(config *configStruct) error {
	if len(config.Targets) == 0 {
		return errors.New("no monitoring targets specified - at least one target is required")
	}

	for i, target := range config.Targets {
		if target.Host == "" {
			return errors.Errorf("target %d: host is required", i)
		}
		if target.Name == "" {
			return errors.Errorf("target %d: name is required", i)
		}
	}

	hasInfluxV1 := config.InfluxDBv1.Addr != ""
	hasInfluxV2 := config.InfluxDBv2.Addr != ""

	if !hasInfluxV1 && !hasInfluxV2 {
		return errors.New("no InfluxDB configuration found - either InfluxDBv1 or InfluxDBv2 must be configured")
	}

	if hasInfluxV1 {
		if config.InfluxDBv1.Database == "" {
			return errors.New("InfluxDBv1.Database is required when InfluxDBv1.Addr is specified")
		}
	}

	if hasInfluxV2 {
		if config.InfluxDBv2.Org == "" {
			return errors.New("InfluxDBv2.Org is required when InfluxDBv2.Addr is specified")
		}
		if config.InfluxDBv2.Bucket == "" {
			return errors.New("InfluxDBv2.Bucket is required when InfluxDBv2.Addr is specified")
		}
		if config.InfluxDBv2.Token == "" {
			return errors.New("InfluxDBv2.Token is required when InfluxDBv2.Addr is specified")
		}
	}

	if config.TimeoutMinutes < 1 || config.TimeoutMinutes > 1440 { // 1 minute to 24 hours
		return errors.Errorf("TimeoutMinutes must be between 1 and 1440 (24 hours), got %d", config.TimeoutMinutes)
	}

	if config.Hostname == "" {
		config.Hostname = "localhost"
		logrus.Warn("hostname not specified in config, using default 'localhost'")
	}

	return nil
}

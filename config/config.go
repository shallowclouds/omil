package config

import (
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
	})
	return config
}

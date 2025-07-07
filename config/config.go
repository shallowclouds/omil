package config

import (
	"fmt"
	"sync"

	"github.com/jinzhu/configor"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Target struct {
	Host string `yaml:"Host"`
	Name string `yaml:"Name"`
}

type ConfigStruct struct {
	mu       sync.RWMutex
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

type configStruct = ConfigStruct

var (
	globalConfig     *ConfigStruct
	globalConfigOnce sync.Once
	globalFilePath   string
)

func SetConfigFilePath(filepath string) {
	globalFilePath = filepath
}

func Config() *ConfigStruct {
	globalConfigOnce.Do(func() {
		var err error
		globalConfig, err = LoadConfig(globalFilePath, "conf/config.yml")
		if err != nil {
			logrus.WithError(err).Fatal("failed to load config from file")
		}
	})
	return globalConfig
}

func LoadConfig(filePaths ...string) (*ConfigStruct, error) {
	config := &ConfigStruct{}
	if err := configor.Load(config, filePaths...); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *ConfigStruct) GetHostname() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Hostname
}

func (c *ConfigStruct) GetTargets() []Target {
	c.mu.RLock()
	defer c.mu.RUnlock()
	targets := make([]Target, len(c.Targets))
	copy(targets, c.Targets)
	return targets
}

func (c *ConfigStruct) GetInfluxDBv2() struct {
	Addr   string `yaml:"Addr"`
	Org    string `yaml:"Org"`
	Bucket string `yaml:"Bucket"`
	Token  string `yaml:"Token"`
} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.InfluxDBv2
}

func (c *ConfigStruct) Validate() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Hostname == "" {
		return errors.New("hostname is required")
	}

	if len(c.Targets) == 0 {
		return errors.New("at least one target is required")
	}

	for i, target := range c.Targets {
		if target.Host == "" {
			return fmt.Errorf("target[%d]: host is required", i)
		}
		if target.Name == "" {
			return fmt.Errorf("target[%d]: name is required", i)
		}
	}

	if c.InfluxDBv2.Addr == "" {
		return errors.New("InfluxDBv2.Addr is required")
	}
	if c.InfluxDBv2.Org == "" {
		return errors.New("InfluxDBv2.Org is required")
	}
	if c.InfluxDBv2.Bucket == "" {
		return errors.New("InfluxDBv2.Bucket is required")
	}
	if c.InfluxDBv2.Token == "" {
		return errors.New("InfluxDBv2.Token is required")
	}

	return nil
}

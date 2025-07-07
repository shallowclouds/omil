package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		name        string
		configYAML  string
		expectError bool
	}{
		{
			name: "valid config",
			configYAML: `
Hostname: "test-host"
InfluxDBv2:
  Addr: "http://localhost:8086"
  Org: "test-org"
  Bucket: "test-bucket"
  Token: "test-token"
Targets:
  - Host: "8.8.8.8"
    Name: "google-dns"
  - Host: "1.1.1.1"
    Name: "cloudflare-dns"
`,
			expectError: false,
		},
		{
			name: "invalid yaml",
			configYAML: `
invalid: yaml: content
  - missing
`,
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.yml")

			err := os.WriteFile(configFile, []byte(testCase.configYAML), 0644)
			require.NoError(t, err)

			config, err := LoadConfig(configFile)

			if testCase.expectError {
				require.Error(t, err)
				require.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				require.Equal(t, "test-host", config.GetHostname())
				targets := config.GetTargets()
				require.Len(t, targets, 2)
				require.Equal(t, "8.8.8.8", targets[0].Host)
				require.Equal(t, "google-dns", targets[0].Name)
			}
		})
	}
}

func TestConfigStruct_Validate(t *testing.T) {
	testCases := []struct {
		name        string
		config      *ConfigStruct
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &ConfigStruct{
				Hostname: "test-host",
				InfluxDBv2: struct {
					Addr   string `yaml:"Addr"`
					Org    string `yaml:"Org"`
					Bucket string `yaml:"Bucket"`
					Token  string `yaml:"Token"`
				}{
					Addr:   "http://localhost:8086",
					Org:    "test-org",
					Bucket: "test-bucket",
					Token:  "test-token",
				},
				Targets: []Target{
					{Host: "8.8.8.8", Name: "google-dns"},
				},
			},
			expectError: false,
		},
		{
			name: "missing hostname",
			config: &ConfigStruct{
				InfluxDBv2: struct {
					Addr   string `yaml:"Addr"`
					Org    string `yaml:"Org"`
					Bucket string `yaml:"Bucket"`
					Token  string `yaml:"Token"`
				}{
					Addr:   "http://localhost:8086",
					Org:    "test-org",
					Bucket: "test-bucket",
					Token:  "test-token",
				},
				Targets: []Target{
					{Host: "8.8.8.8", Name: "google-dns"},
				},
			},
			expectError: true,
			errorMsg:    "hostname is required",
		},
		{
			name: "no targets",
			config: &ConfigStruct{
				Hostname: "test-host",
				InfluxDBv2: struct {
					Addr   string `yaml:"Addr"`
					Org    string `yaml:"Org"`
					Bucket string `yaml:"Bucket"`
					Token  string `yaml:"Token"`
				}{
					Addr:   "http://localhost:8086",
					Org:    "test-org",
					Bucket: "test-bucket",
					Token:  "test-token",
				},
				Targets: []Target{},
			},
			expectError: true,
			errorMsg:    "at least one target is required",
		},
		{
			name: "target missing host",
			config: &ConfigStruct{
				Hostname: "test-host",
				InfluxDBv2: struct {
					Addr   string `yaml:"Addr"`
					Org    string `yaml:"Org"`
					Bucket string `yaml:"Bucket"`
					Token  string `yaml:"Token"`
				}{
					Addr:   "http://localhost:8086",
					Org:    "test-org",
					Bucket: "test-bucket",
					Token:  "test-token",
				},
				Targets: []Target{
					{Name: "test-target"},
				},
			},
			expectError: true,
			errorMsg:    "target[0]: host is required",
		},
		{
			name: "missing influxdb addr",
			config: &ConfigStruct{
				Hostname: "test-host",
				InfluxDBv2: struct {
					Addr   string `yaml:"Addr"`
					Org    string `yaml:"Org"`
					Bucket string `yaml:"Bucket"`
					Token  string `yaml:"Token"`
				}{
					Org:    "test-org",
					Bucket: "test-bucket",
					Token:  "test-token",
				},
				Targets: []Target{
					{Host: "8.8.8.8", Name: "google-dns"},
				},
			},
			expectError: true,
			errorMsg:    "InfluxDBv2.Addr is required",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := testCase.config.Validate()

			if testCase.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), testCase.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigStruct_GetMethods(t *testing.T) {
	config := &ConfigStruct{
		Hostname: "test-host",
		InfluxDBv2: struct {
			Addr   string `yaml:"Addr"`
			Org    string `yaml:"Org"`
			Bucket string `yaml:"Bucket"`
			Token  string `yaml:"Token"`
		}{
			Addr:   "http://localhost:8086",
			Org:    "test-org",
			Bucket: "test-bucket",
			Token:  "test-token",
		},
		Targets: []Target{
			{Host: "8.8.8.8", Name: "google-dns"},
			{Host: "1.1.1.1", Name: "cloudflare-dns"},
		},
	}

	require.Equal(t, "test-host", config.GetHostname())

	targets := config.GetTargets()
	require.Len(t, targets, 2)
	require.Equal(t, "8.8.8.8", targets[0].Host)
	require.Equal(t, "google-dns", targets[0].Name)

	influxConfig := config.GetInfluxDBv2()
	require.Equal(t, "http://localhost:8086", influxConfig.Addr)
	require.Equal(t, "test-org", influxConfig.Org)
	require.Equal(t, "test-bucket", influxConfig.Bucket)
	require.Equal(t, "test-token", influxConfig.Token)
}

func TestTarget(t *testing.T) {
	target := Target{
		Host: "example.com",
		Name: "test-target",
	}

	require.Equal(t, "example.com", target.Host)
	require.Equal(t, "test-target", target.Name)
}

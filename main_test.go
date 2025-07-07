package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shallowclouds/omil/config"
	"github.com/stretchr/testify/require"
)

func TestNewApp(t *testing.T) {
	tmpDir := t.TempDir()
	fallbackConfigDir := filepath.Join(tmpDir, "conf")
	err := os.MkdirAll(fallbackConfigDir, 0755)
	require.NoError(t, err)

	fallbackConfig := filepath.Join(fallbackConfigDir, "config.yml")
	fallbackYAML := `
Hostname: "fallback-host"
InfluxDBv2:
  Addr: "http://localhost:8086"
  Org: "fallback-org"
  Bucket: "fallback-bucket"
  Token: "fallback-token"
Targets:
  - Host: "127.0.0.1"
    Name: "fallback-target"
`
	err = os.WriteFile(fallbackConfig, []byte(fallbackYAML), 0644)
	require.NoError(t, err)

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	testCases := []struct {
		name        string
		configYAML  string
		expectError bool
		errorMsg    string
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
`,
			expectError: false,
		},
		{
			name: "invalid config - missing hostname",
			configYAML: `
InfluxDBv2:
  Addr: "http://localhost:8086"
  Org: "test-org"
  Bucket: "test-bucket"
  Token: "test-token"
Targets:
  - Host: "8.8.8.8"
    Name: "google-dns"
`,
			expectError: true,
			errorMsg:    "hostname is required",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			configFile := filepath.Join(tmpDir, "test-config.yml")

			err := os.WriteFile(configFile, []byte(testCase.configYAML), 0644)
			require.NoError(t, err)

			conf, err := config.LoadConfig(configFile)
			if testCase.expectError {
				if err == nil {
					err = conf.Validate()
				}
				require.Error(t, err)
				if testCase.errorMsg != "" {
					require.Contains(t, err.Error(), testCase.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, conf)
				require.NoError(t, conf.Validate())

				app, err := NewApp(configFile)
				require.NoError(t, err)
				require.NotNil(t, app)
				require.NotNil(t, app.config)
				require.NotNil(t, app.metricClient)
			}
		})
	}
}

func TestApp_CreateMonitors(t *testing.T) {
	tmpDir := t.TempDir()
	fallbackConfigDir := filepath.Join(tmpDir, "conf")
	err := os.MkdirAll(fallbackConfigDir, 0755)
	require.NoError(t, err)

	fallbackConfig := filepath.Join(fallbackConfigDir, "config.yml")
	fallbackYAML := `
Hostname: "fallback-host"
InfluxDBv2:
  Addr: "http://localhost:8086"
  Org: "fallback-org"
  Bucket: "fallback-bucket"
  Token: "fallback-token"
Targets:
  - Host: "127.0.0.1"
    Name: "fallback-target"
`
	err = os.WriteFile(fallbackConfig, []byte(fallbackYAML), 0644)
	require.NoError(t, err)

	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)

	configFile := filepath.Join(tmpDir, "test-config.yml")
	configYAML := `
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
`

	err = os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(t, err)

	app, err := NewApp(configFile)
	require.NoError(t, err)

	monitors, err := app.CreateMonitors()
	require.NoError(t, err)
	require.True(t, len(monitors) >= 1, "Expected at least 1 monitor, got %d", len(monitors))
	require.True(t, len(monitors) <= 2, "Expected at most 2 monitors, got %d", len(monitors))

	for _, monitor := range monitors {
		require.Contains(t, monitor.Name(), "test-host")
		require.True(t,
			monitor.Name() == "<test-host>-<google-dns>" ||
				monitor.Name() == "<test-host>-<cloudflare-dns>",
			"Unexpected monitor name: %s", monitor.Name())
	}
}

func TestApp_Run_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yml")

	configYAML := `
Hostname: "test-host"
InfluxDBv2:
  Addr: "http://localhost:8086"
  Org: "test-org"
  Bucket: "test-bucket"
  Token: "test-token"
Targets:
  - Host: "127.0.0.1"
    Name: "localhost"
`

	err := os.WriteFile(configFile, []byte(configYAML), 0644)
	require.NoError(t, err)

	app, err := NewApp(configFile)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	err = app.Run(ctx)
	require.NoError(t, err)
}

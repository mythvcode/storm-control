package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testConfig = `
---
# Logger options.
logger:
  file: "test_file"
  level: info
watcher:
  block_delay: 123
  block_enabled: true
  block_threshold: 555
  device_list:
  - eth5
  - eth55 
  device_regex: test_regex
exporter:
  enable: false
  enable_request_logging: false
  enable_runtime_metrics: true
  server_address: yaml_test_address
  server_port:    1010
  request_timeout: 55555
  telemetry_path: "/test_conf_path"

`

func testDefaults(t *testing.T, cfg StormControlConfig) {
	t.Helper()
	require.Empty(t, cfg.Logger.File)
	require.Equal(t, "debug", cfg.Logger.Level)
	require.Equal(t, 10, cfg.Watcher.BlockDelay)
	require.Equal(t, uint64(100), cfg.Watcher.BlockThreshold)
	require.Equal(t, `^tap.{8}-.{2}$`, cfg.Watcher.DevRegEx)
	require.False(t, cfg.Watcher.BlockEnabled)
	require.Empty(t, cfg.Watcher.StaticDevList)
	require.True(t, cfg.Exporter.Enable)
	require.True(t, cfg.Exporter.EnableRequestLogging)
	require.False(t, cfg.Exporter.EnableRuntimeMetrics)
	require.Equal(t, "localhost", cfg.Exporter.ServerAddress)
	require.Equal(t, 8080, cfg.Exporter.ServerPort)
	require.Equal(t, 10, cfg.Exporter.RequestTimeout)
	require.Equal(t, "/metrics", cfg.Exporter.TelemetryPath)
}

func setEnvVars(t *testing.T) {
	t.Helper()
	envVars := []struct {
		envName string
		value   string
	}{
		{
			"LOG_LEVEL",
			"env_log_level",
		},
		{
			"LOG_FILE",
			"env_log_file",
		},
		{
			"BLOCK_DELAY",
			"12345",
		},
		{
			"BLOCK_ENABLED",
			"true",
		},
		{
			"BLOCK_THRESHOLD",
			"55555",
		},
		{
			"STATIC_DEV_LIST",
			"eth1, eth2",
		},
		{
			"DEV_REGEX",
			"test_env_regexp",
		},
		{
			"EXPORTER_HOST",
			"test_host",
		},
		{
			"EXPORTER_PORT",
			"12345",
		},
		{
			"EXPORTER_REQUEST_TIMEOUT",
			"11111",
		},
		{
			"EXPORTER_TELEMETRY_PATH",
			"/test_path",
		},
		{
			"EXPORTER_ENABLE",
			"false",
		},
		{
			"EXPORTER_ENABLE_REQUEST_LOGGING",
			"false",
		},
		{
			"EXPORTER_ENABLE_RUNTIME_METRICS",
			"true",
		},
	}
	for _, env := range envVars {
		t.Setenv(env.envName, env.value)
	}

	t.Cleanup(func() {
		for _, env := range envVars {
			require.NoError(t, os.Unsetenv(env.envName))
		}
	})
}

func TestCheckDefaults(t *testing.T) {
	cfg, err := loadFromBytes([]byte{})
	require.NoError(t, err)
	testDefaults(t, cfg)
}

func TestLoadFromEnv(t *testing.T) {
	cfg, err := ReadConfig("")
	require.NoError(t, err)
	testDefaults(t, cfg)

	setEnvVars(t)
	cfg, err = ReadConfig("")
	require.NoError(t, err)

	require.Equal(t, "env_log_file", cfg.Logger.File)
	require.Equal(t, "env_log_level", cfg.Logger.Level)
	require.Equal(t, 12345, cfg.Watcher.BlockDelay)
	require.True(t, cfg.Watcher.BlockEnabled)
	require.Equal(t, uint64(55555), cfg.Watcher.BlockThreshold)
	require.Equal(t, "test_env_regexp", cfg.Watcher.DevRegEx)
	require.Equal(t, []string{"eth1", "eth2"}, cfg.Watcher.StaticDevList)
	require.False(t, cfg.Exporter.Enable)
	require.False(t, cfg.Exporter.EnableRequestLogging)
	require.True(t, cfg.Exporter.EnableRuntimeMetrics)
	require.Equal(t, "test_host", cfg.Exporter.ServerAddress)
	require.Equal(t, 12345, cfg.Exporter.ServerPort)
	require.Equal(t, 11111, cfg.Exporter.RequestTimeout)
	require.Equal(t, "/test_path", cfg.Exporter.TelemetryPath)
}

func TestLoadFromFile(t *testing.T) {
	cfg, err := loadFromBytes([]byte(testConfig))
	require.NoError(t, err)
	require.Equal(t, "test_file", cfg.Logger.File)
	require.Equal(t, "info", cfg.Logger.Level)
	require.Equal(t, 123, cfg.Watcher.BlockDelay)
	require.True(t, cfg.Watcher.BlockEnabled)
	require.Equal(t, uint64(555), cfg.Watcher.BlockThreshold)
	require.Equal(t, "test_regex", cfg.Watcher.DevRegEx)
	require.Equal(t, []string{"eth5", "eth55"}, cfg.Watcher.StaticDevList)
	require.False(t, cfg.Exporter.Enable)
	require.False(t, cfg.Exporter.EnableRequestLogging)
	require.True(t, cfg.Exporter.EnableRuntimeMetrics)
	require.Equal(t, "yaml_test_address", cfg.Exporter.ServerAddress)
	require.Equal(t, 1010, cfg.Exporter.ServerPort)
	require.Equal(t, 55555, cfg.Exporter.RequestTimeout)
	require.Equal(t, "/test_conf_path", cfg.Exporter.TelemetryPath)
}

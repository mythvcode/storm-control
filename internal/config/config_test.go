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
`

func testDefaults(t *testing.T, cfg StormControlConfig) {
	t.Helper()
	require.Empty(t, cfg.Logger.File)
	require.Equal(t, "debug", cfg.Logger.Level)
	require.Equal(t, 0, cfg.Watcher.BlockDelay)
	require.Equal(t, uint64(10), cfg.Watcher.BlockThreshold)
	require.Equal(t, `^tap.{8}-.{2}$`, cfg.Watcher.DevRegEx)
	require.False(t, cfg.Watcher.BlockEnabled)
	require.Empty(t, cfg.Watcher.StaticDevList)
}

func setEnvVars(t *testing.T) func() {
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
	}
	for _, env := range envVars {
		require.NoError(t, os.Setenv(env.envName, env.value))
	}

	return func() {
		for _, env := range envVars {
			require.NoError(t, os.Unsetenv(env.envName))
		}
	}
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

	unsetEnv := setEnvVars(t)
	defer unsetEnv()
	cfg, err = ReadConfig("")
	require.NoError(t, err)

	require.Equal(t, "env_log_file", cfg.Logger.File)
	require.Equal(t, "env_log_level", cfg.Logger.Level)
	require.Equal(t, 12345, cfg.Watcher.BlockDelay)
	require.True(t, cfg.Watcher.BlockEnabled)
	require.Equal(t, uint64(55555), cfg.Watcher.BlockThreshold)
	require.Equal(t, "test_env_regexp", cfg.Watcher.DevRegEx)
	require.Equal(t, []string{"eth1", "eth2"}, cfg.Watcher.StaticDevList)
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
}

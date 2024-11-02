package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/creasty/defaults"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

type StormControlConfig struct {
	Watcher  WatcherConfig `yaml:"watcher"`
	Logger   LoggerConfig  `yaml:"logger"`
	Exporter Exporter      `yaml:"exporter"`
}

type LoggerConfig struct {
	Level string `default:"debug" env:"LOG_LEVEL" yaml:"level"`
	File  string `default:""      env:"LOG_FILE"  yaml:"file"`
}

type WatcherConfig struct {
	BlockDelay     int      `default:"10"             env:"BLOCK_DELAY"     yaml:"block_delay"`
	BlockEnabled   bool     `default:"false"          env:"BLOCK_ENABLED"   yaml:"block_enabled"`
	BlockThreshold uint64   `default:"100"            env:"BLOCK_THRESHOLD" yaml:"block_threshold"`
	StaticDevList  []string `default:"[]"             env:"STATIC_DEV_LIST" yaml:"device_list"`
	DevRegEx       string   `default:"^tap.{8}-.{2}$" env:"DEV_REGEX"       yaml:"device_regex"`
}

type Exporter struct {
	ServerAddress        string `default:"localhost" env:"EXPORTER_HOST"                   yaml:"server_address"`
	ServerPort           int    `default:"8080"      env:"EXPORTER_PORT"                   yaml:"server_port"`
	RequestTimeout       int    `default:"10"        env:"EXPORTER_REQUEST_TIMEOUT"        yaml:"request_timeout"`
	TelemetryPath        string `default:"/metrics"  env:"EXPORTER_TELEMETRY_PATH"         yaml:"telemetry_path"`
	Enable               bool   `default:"true"      env:"EXPORTER_ENABLE"                 yaml:"enable"`
	EnableRequestLogging bool   `default:"true"      env:"EXPORTER_ENABLE_REQUEST_LOGGING" yaml:"enable_request_logging"`
	EnableRuntimeMetrics bool   `default:"false"     env:"EXPORTER_ENABLE_RUNTIME_METRICS" yaml:"enable_runtime_metrics"`
}

func (c *StormControlConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain StormControlConfig

	return unmarshal((*plain)(c))
}

func ReadEnv(cfg StormControlConfig) (StormControlConfig, error) {
	err := envconfig.Process(
		context.Background(),
		&envconfig.Config{DefaultOverwrite: true, Target: &cfg},
	)

	return cfg, err
}

func ReadConfig(file string) (StormControlConfig, error) {
	if file == "" {
		return ReadEnv(getDefault())
	}

	return loadFromFile(file)
}

func loadFromFile(file string) (StormControlConfig, error) {
	configBytes, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return StormControlConfig{}, fmt.Errorf("unable to read config: %w", err)
	}

	return loadFromBytes(configBytes)
}

func getDefault() StormControlConfig {
	res, _ := loadFromBytes([]byte{})

	return res
}

func loadFromBytes(data []byte) (StormControlConfig, error) {
	var config StormControlConfig

	// make empty config for defaults package to call function UnmarshalYAML
	if len(data) == 0 {
		data = []byte("watcher:")
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return StormControlConfig{}, fmt.Errorf("unable to unmarshal config: %w", err)
	}

	return config, nil
}

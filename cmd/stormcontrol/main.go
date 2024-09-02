package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/mythvcode/storm-control/internal/config"
	"github.com/mythvcode/storm-control/internal/logger"
	"github.com/mythvcode/storm-control/internal/watcher"
)

var cfgPath string

func init() {
	flag.StringVar(&cfgPath, "config", "", "Path to config file")
}

func main() {
	flag.Parse()
	cfg, err := config.ReadConfig(cfgPath)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	if err != nil {
		if cfgPath != "" {
			logger.Default().Errorf("Error read config file from file %s: %s", cfgPath, err.Error())
		} else {
			logger.Default().Errorf("Error read config: %s", err.Error())
		}
		os.Exit(1)
	}
	if err := logger.Init(cfg.Logger.File, cfg.Logger.Level); err != nil {
		logger.Default().Errorf("Error init logger: %s", err.Error())
		os.Exit(1)
	}
	netWatcher, err := watcher.New(cfg)
	if err != nil {
		logger.GetLogger().Errorf("Error create watcher: %s", err.Error())
		os.Exit(1)
	}
	defer netWatcher.Stop()

	go func() {
		netWatcher.Start()
	}()
	<-sigs
}

package main

import (
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mythvcode/storm-control/internal/config"
	"github.com/mythvcode/storm-control/internal/ebpfloader"
	"github.com/mythvcode/storm-control/internal/exporter"
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
	eBPFProg, err := ebpfloader.LoadCollection()
	if err != nil {
		logger.GetLogger().Errorf("Error load eBPF program %s", err.Error())
		os.Exit(1)
	}
	netWatcher, err := watcher.New(cfg, eBPFProg)
	if err != nil {
		logger.GetLogger().Errorf("Error create watcher: %s", err.Error())
		os.Exit(1)
	}

	if cfg.Exporter.Enable {
		exporter, err := exporter.New(cfg.Exporter, eBPFProg)
		if err != nil {
			logger.GetLogger().Errorf("Error start exporter: %s", err.Error())
			os.Exit(1)
		}
		started := make(chan error)
		go func() {
			started <- exporter.Start()
		}()
		go func() {
			if err := <-started; err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.GetLogger().Errorf("Error start exporter: %s", err.Error())
				os.Exit(1)
			}
		}()

		defer exporter.Stop()
	}

	defer netWatcher.Stop()

	go func() {
		netWatcher.Start()
	}()
	<-sigs
}

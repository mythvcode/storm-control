package watcher

import (
	"log/slog"
	"net"
	"regexp"
	"time"

	"github.com/cilium/ebpf"
	"github.com/mythvcode/storm-control/internal/config"
	"github.com/mythvcode/storm-control/internal/logger"
)

type eBPFProg interface {
	AttachXDPToNetDevice(ndev int) error
	ForceDetachXDP(ndev int)
	GetStatsMap() *ebpf.Map
	GetDropMap() *ebpf.Map
	Close()
}
type Watcher struct {
	devWatcherMap map[int]*netDevWatcher
	ebpfProg      eBPFProg
	config        config.WatcherConfig
	closed        chan struct{}
	netDevReg     *regexp.Regexp
	log           *logger.Logger
}

var listInterfaces = net.Interfaces

func isDevExist(devLis []net.Interface, index int) bool {
	for _, dev := range devLis {
		if dev.Index == index {
			return true
		}
	}

	return false
}

func New(cfg config.StormControlConfig, prog eBPFProg) (*Watcher, error) {
	regExp, err := regexp.Compile(cfg.Watcher.DevRegEx)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		devWatcherMap: make(map[int]*netDevWatcher),
		ebpfProg:      prog,
		config:        cfg.Watcher,
		netDevReg:     regExp,
		closed:        make(chan struct{}),
		log:           logger.GetLogger().With(slog.String(logger.Component, "Watcher")),
	}, nil
}

func (w *Watcher) makeNetDevWatcher(netDev int, netDevName string) *netDevWatcher {
	return newNetDevWatcher(
		netDev,
		netDevName,
		w.config.BlockThreshold,
		time.Duration(w.config.BlockDelay)*time.Second,
		w.ebpfProg.GetStatsMap(),
		w.ebpfProg.GetDropMap(),
	)
}

func (w *Watcher) findStaticNetDevices(allNetDevices []net.Interface) []net.Interface {
	result := make([]net.Interface, 0, 1)
	for _, netDev := range allNetDevices {
		for _, staticNetDevName := range w.config.StaticDevList {
			if netDev.Name == staticNetDevName {
				result = append(result, netDev)
			}
		}
	}

	return result
}

func (w *Watcher) getNetDevicesForAttach() ([]net.Interface, error) {
	var result []net.Interface
	allNetDevs, err := listInterfaces()
	if err != nil {
		return result, err
	}
	if len(w.config.StaticDevList) == 0 {
		for _, netDev := range allNetDevs {
			if w.netDevReg.MatchString(netDev.Name) {
				result = append(result, netDev)
			}
		}
	} else {
		result = w.findStaticNetDevices(allNetDevs)
	}

	return result, nil
}

func (w *Watcher) findAndAttachNetDev() {
	netDevices, err := w.getNetDevicesForAttach()
	if err != nil {
		w.log.Errorf("Error get netDevList list: %s", err.Error())

		return
	}
	for _, nDev := range netDevices {
		if _, ok := w.devWatcherMap[nDev.Index]; !ok {
			w.log.Infof("Attach program to %s (%d)", nDev.Name, nDev.Index)
			if err := w.ebpfProg.AttachXDPToNetDevice(nDev.Index); err != nil {
				w.log.Errorf("Error attach program to device %d %s %s", nDev.Index, nDev.Name, err.Error())

				continue
			}
			nDevWatcher := w.makeNetDevWatcher(nDev.Index, nDev.Name)
			w.devWatcherMap[nDev.Index] = nDevWatcher
			// do not start net device watcher process in case drop action disabled
			if w.config.BlockEnabled {
				go nDevWatcher.startWatching()
			}
		}
	}
}

func (w *Watcher) cleanNetDev() {
	allNetDev, err := listInterfaces()
	if err != nil {
		w.log.Errorf("Error get network interfaces %s", err.Error())

		return
	}
	for _, devWatcher := range w.devWatcherMap {
		if !isDevExist(allNetDev, devWatcher.index()) {
			w.log.Infof("Interface %s not found stop watch process", devWatcher.devInfo())
			devWatcher.stop()
			delete(w.devWatcherMap, devWatcher.index())
			w.ebpfProg.ForceDetachXDP(devWatcher.index())
		}
	}
}

func (w *Watcher) startDynamicWatcher() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-w.closed:
			return
		case <-ticker.C:
			w.findAndAttachNetDev()
			w.cleanNetDev()
		}
	}
}

func (w *Watcher) Start() {
	w.log.Infof("Start device watcher drop")
	if !w.config.BlockEnabled {
		w.log.Warningf("Block action disabled!")
	}
	w.startDynamicWatcher()
}

func (w *Watcher) Stop() {
	w.log.Infof("Stop device watcher")
	close(w.closed)
	w.StopDevWatchers()
	w.ebpfProg.Close()
}

func (w *Watcher) StopDevWatchers() {
	for _, devWatcher := range w.devWatcherMap {
		devWatcher.stop()
		w.ebpfProg.ForceDetachXDP(devWatcher.index())
	}
}

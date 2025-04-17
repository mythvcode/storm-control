package watcher

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mythvcode/storm-control/internal/ebpfloader"
	"github.com/mythvcode/storm-control/internal/logger"
)

const (
	_ = iota
	broadcastType
	ipv4McastType
	ipv6McastType
	otherType
)

const (
	unblockAction = 1
	blockAction   = 2
)

type netDevWatcher struct {
	netDevIndex      int
	netDevName       string
	blockThreshold   uint64
	unblockThreshold uint64
	dropDelay        time.Duration
	stopChan         chan struct{}
	ebpfProg         eBPFProg
	dropMapMux       sync.Mutex

	dropState dropStateConfig
	log       *logger.Logger
}

type dropStateConfig struct {
	brDropped        atomic.Bool
	ipv4McastDropped atomic.Bool
	ipv6McastDropped atomic.Bool
	other            atomic.Bool
}

type updateDropConfig struct {
	br    uint8
	ipv4  uint8
	ipv6  uint8
	other uint8
}

// in ebpf kernel module 0 is pass 1 is block
func getEBPFAction(configValue uint8) uint8 {
	if configValue == blockAction {
		return 1
	}

	return 0
}

func (u *updateDropConfig) isEmpty() bool {
	return u.br == 0 &&
		u.ipv4 == 0 &&
		u.ipv6 == 0 &&
		u.other == 0
}

// Creates Interface watcher instance.
// Map entries to this interface must be created before start watching process
func newNetDevWatcher(
	netDev int,
	netDevName string,
	blockThreshold uint64,
	dropDelay time.Duration,
	ebpfProg eBPFProg,
) *netDevWatcher {
	return &netDevWatcher{
		netDevIndex:      netDev,
		netDevName:       netDevName,
		blockThreshold:   blockThreshold,
		unblockThreshold: blockThreshold * 3,
		dropDelay:        dropDelay,
		stopChan:         make(chan struct{}),
		ebpfProg:         ebpfProg,
		log:              logger.GetLogger().With(slog.String(logger.Component, "NetDevWatcher")),
	}
}

func (n *netDevWatcher) stop() {
	close(n.stopChan)
}

func (n *netDevWatcher) index() int {
	return n.netDevIndex
}

func (n *netDevWatcher) devInfo() string {
	return fmt.Sprintf("%s (%d)", n.netDevName, n.netDevIndex)
}

func (n *netDevWatcher) getStats() (ebpfloader.PacketCounter, error) {
	return n.ebpfProg.GetDevStat(n.index())
}

func (n *netDevWatcher) startUnblockWatcher(update updateDropConfig) {
	if update.br != 0 {
		go n.watchUnblock(broadcastType)
	}
	if update.ipv4 != 0 {
		go n.watchUnblock(ipv4McastType)
	}
	if update.ipv6 != 0 {
		go n.watchUnblock(ipv6McastType)
	}
	if update.other != 0 {
		go n.watchUnblock(otherType)
	}
}

func (n *netDevWatcher) acquireBlockState(trafType int) bool {
	switch trafType {
	case broadcastType:
		if n.dropState.brDropped.CompareAndSwap(false, true) {
			return true
		}

	case ipv4McastType:
		if n.dropState.ipv4McastDropped.CompareAndSwap(false, true) {
			return true
		}

	case ipv6McastType:
		if n.dropState.ipv6McastDropped.CompareAndSwap(false, true) {
			return true
		}
	case otherType:
		if n.dropState.other.CompareAndSwap(false, true) {
			return true
		}
	}

	return false
}

func (n *netDevWatcher) releaseBlockState(trafType int) {
	switch trafType {
	case broadcastType:
		n.dropState.brDropped.Store(false)

	case ipv4McastType:
		n.dropState.ipv4McastDropped.Store(false)

	case ipv6McastType:
		n.dropState.ipv6McastDropped.Store(false)
	case otherType:
		n.dropState.other.Store(false)
	}
}

func (n *netDevWatcher) checkAndUnblock(prevStats, curState *ebpfloader.PacketCounter, trafType int) (bool, error) { //nolint
	switch trafType {
	case broadcastType:
		if (curState.Broadcast.Dropped - prevStats.Broadcast.Dropped) < n.unblockThreshold {
			if err := n.updateDropMap(updateDropConfig{br: unblockAction}); err != nil {
				return false, err
			}
			n.log.Debugf("Unblock broadcast traffic dev: %s", n.devInfo())

			return true, nil
		}

	case ipv4McastType:
		if (curState.IPv4MCast.Dropped - prevStats.IPv4MCast.Dropped) < n.unblockThreshold {
			if err := n.updateDropMap(updateDropConfig{ipv4: unblockAction}); err != nil {
				return false, err
			}
			n.log.Debugf("Unblock IPv4 Multicast traffic dev: %s", n.devInfo())

			return true, nil
		}

	case ipv6McastType:
		if (curState.IPv6MCast.Dropped - prevStats.IPv6MCast.Dropped) < n.unblockThreshold {
			if err := n.updateDropMap(updateDropConfig{ipv6: unblockAction}); err != nil {
				return false, err
			}
			n.log.Debugf("Unblock IPv6 Multicast traffic dev: %s", n.devInfo())

			return true, nil
		}
	case otherType:
		if (curState.OtherMcast.Dropped - prevStats.OtherMcast.Dropped) < n.unblockThreshold {
			if err := n.updateDropMap(updateDropConfig{other: unblockAction}); err != nil {
				return false, err
			}
			n.log.Debugf("Unblock other Multicast traffic dev: %s", n.devInfo())

			return true, nil
		}
	}

	return false, nil
}

// async function for drop packet calculation for specific type of traffic
// calculates statistic every 3 seconds and make unblock decisions
func (n *netDevWatcher) watchUnblock(trafType int) {
	if !n.acquireBlockState(trafType) {
		return
	}
	defer n.releaseBlockState(trafType)

	timer := time.NewTimer(n.dropDelay)
	defer timer.Stop()
	select {
	case <-n.stopChan:
		return
	case <-timer.C:
	}

	prevStats, err := n.getStats()
	if err != nil {
		n.log.Errorf("Error get statistics for interface %s: %s", n.devInfo(), err.Error())
	}
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()
	for {
		select {
		case <-n.stopChan:
			n.log.Debugf("Stop watching command received for interface %s, stop watching.", n.devInfo())

			return
		case <-ticker.C:
			stats, err := n.getStats()
			if err != nil {
				n.log.Errorf("Error get statistics for interface %s: %s", n.devInfo(), err.Error())

				continue
			}
			ok, err := n.checkAndUnblock(&prevStats, &stats, trafType)
			if err != nil {
				n.log.Errorf("Error check unblock status for interface %s", n.devInfo())
			}
			if ok {
				return
			}
			prevStats = stats
		}
	}
}

func (n *netDevWatcher) updateDropMap(update updateDropConfig) error {
	if update.isEmpty() {
		return nil
	}
	n.dropMapMux.Lock()
	defer n.dropMapMux.Unlock()
	result, err := n.ebpfProg.GetDevDropCfg(n.netDevIndex)
	if err != nil {
		return err
	}

	if update.br != 0 {
		result.Broadcast = getEBPFAction(update.br)
	}
	if update.ipv4 != 0 {
		result.IPv4MCast = getEBPFAction(update.ipv4)
	}
	if update.ipv6 != 0 {
		result.IPv6MCast = getEBPFAction(update.ipv6)
	}
	if update.other != 0 {
		result.Multicast = getEBPFAction(update.other)
	}

	return n.ebpfProg.UpdateDevDropCfg(n.netDevIndex, result)
}

func (n *netDevWatcher) getCalculateStatsFuc() func(statStruct ebpfloader.PacketCounter) updateDropConfig {
	var stats ebpfloader.PacketCounter

	return func(curStats ebpfloader.PacketCounter) updateDropConfig {
		blockStruct := updateDropConfig{}
		if (curStats.Broadcast.Passed - stats.Broadcast.Passed) > n.blockThreshold {
			n.log.Debugf("Block broadcast traffic %s", n.devInfo())
			blockStruct.br = blockAction
		}
		if (curStats.IPv4MCast.Passed - stats.IPv4MCast.Passed) > n.blockThreshold {
			n.log.Debugf("Block IPv4 multicast traffic %s", n.devInfo())
			blockStruct.ipv4 = blockAction
		}
		if (curStats.IPv6MCast.Passed - stats.IPv6MCast.Passed) > n.blockThreshold {
			n.log.Debugf("Block IPv6 multicast traffic %s", n.devInfo())
			blockStruct.ipv6 = blockAction
		}

		if (curStats.OtherMcast.Passed - stats.OtherMcast.Passed) > n.blockThreshold {
			n.log.Debugf("Block other multicast traffic %s", n.devInfo())
			blockStruct.other = blockAction
		}
		stats = curStats

		return blockStruct
	}
}

// async function started in separate goroutine
// calculates statistic every second and make block decisions
// only one instance of this function must be launched for  specific interface
func (n *netDevWatcher) startWatching() {
	calculateState := n.getCalculateStatsFuc()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopChan:
			n.log.Debugf("Stop watching command received for interface %s, stop watching.", n.devInfo())

			return

		case <-ticker.C:
			stats, err := n.getStats()
			if err != nil {
				n.log.Errorf("Error get statistic for interface %s %s", n.devInfo(), err.Error())

				continue
			}
			dropConf := calculateState(stats)
			if !dropConf.isEmpty() {
				if err := n.updateDropMap(dropConf); err != nil {
					n.log.Errorf("Error block traffic on interface %s: caused %s", n.devInfo(), err.Error())

					continue
				}
				n.startUnblockWatcher(dropConf)
			}
		}
	}
}

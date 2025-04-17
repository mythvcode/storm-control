package ebpfloader

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/cilium/ebpf/link"
)

type EbfProgram struct {
	Collection *collection
	lMux       sync.Mutex
	Links      map[int]link.Link
}

func toUint32(interfaceIndex int) (uint32, error) {
	if interfaceIndex < 0 {
		return 0, fmt.Errorf("negative interface index %d", interfaceIndex)
	}
	if interfaceIndex > math.MaxUint32 {
		return 0, fmt.Errorf("overflow interface uint32 interface index %d, max value %d", interfaceIndex, math.MaxUint32)
	}

	return uint32(interfaceIndex), nil
}

func New() (*EbfProgram, error) {
	prog := &EbfProgram{
		Links: make(map[int]link.Link),
	}
	col, err := loadCollection()
	if err != nil {
		return nil, err
	}
	prog.Collection = col

	return prog, err
}

func (e *EbfProgram) AttachXDP(ndev int) error {
	devIndexUint32, err := toUint32(ndev)
	if err != nil {
		return err
	}
	link, err := link.AttachXDP(
		link.XDPOptions{
			Program:   e.Collection.getProgram(),
			Interface: ndev,
			Flags:     link.XDPGenericMode,
		})
	if err != nil {
		return err
	}

	if err := e.addNetDevToMaps(devIndexUint32); err != nil {
		link.Close() //nolint

		return err
	}

	e.lMux.Lock()
	defer e.lMux.Unlock()
	e.Links[ndev] = link

	return nil
}

func (e *EbfProgram) DetachXDP(ndev int) error {
	devIndexUint32, err := toUint32(ndev)
	if err != nil {
		return err
	}

	if err := e.removeNetDevFromMaps(devIndexUint32); err != nil {
		return err
	}

	xdpLink := e.Links[ndev]
	if xdpLink == nil {
		return fmt.Errorf("xdp is not attached to interface %d", ndev)
	}
	if err := xdpLink.Close(); err != nil {
		return err
	}
	e.lMux.Lock()
	defer e.lMux.Unlock()
	delete(e.Links, ndev)

	return nil
}

func (e *EbfProgram) ForceDetachXDP(ndev int) {
	devIndexUint32, err := toUint32(ndev)
	if err != nil {
		return
	}
	e.removeNetDevFromMaps(devIndexUint32) //nolint
	xdpLink, exist := e.Links[ndev]
	if exist {
		xdpLink.Close()
	}
	e.lMux.Lock()
	defer e.lMux.Unlock()
	delete(e.Links, ndev)
}

func (e *EbfProgram) addNetDevToMaps(ndev uint32) error {
	if err := e.Collection.putStatValue(ndev); err != nil {
		return err
	}

	if err := e.Collection.putDropValue(ndev, DropPKT{}); err != nil {
		if delErr := e.Collection.deleteStatValue(ndev); delErr != nil {
			return errors.Join(err, delErr)
		}

		return err
	}

	return nil
}

func (e *EbfProgram) removeNetDevFromMaps(ndev uint32) error {
	if err := e.Collection.deleteStatValue(ndev); err != nil {
		return err
	}

	if err := e.Collection.deleteDropValue(ndev); err != nil {
		return err
	}

	return nil
}

func (e *EbfProgram) GetStatistic() (Statistic, error) {
	return e.Collection.getStatistic()
}

func (e *EbfProgram) GetDevStat(devIndex int) (PacketCounter, error) {
	devIndexUint32, err := toUint32(devIndex)
	if err != nil {
		return PacketCounter{}, err
	}

	return e.Collection.lookupStatValue(devIndexUint32)
}

func (e *EbfProgram) GetDevDropCfg(devIndex int) (DropPKT, error) {
	devIndexUint32, err := toUint32(devIndex)
	if err != nil {
		return DropPKT{}, err
	}

	return e.Collection.lookupDropValue(devIndexUint32)
}

func (e *EbfProgram) UpdateDevDropCfg(devIndex int, cfg DropPKT) error {
	devIndexUint32, err := toUint32(devIndex)
	if err != nil {
		return err
	}

	return e.Collection.updateDropValue(devIndexUint32, cfg)
}

func (e *EbfProgram) Close() {
	e.lMux.Lock()
	for _, ln := range e.Links {
		ln.Close()
	}
	e.lMux.Unlock()

	e.Collection.Close()
}

package ebpfloader

import (
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/mythvcode/storm-control/ebpfxdp"
)

type (
	CounterStat map[uint32]ebpfxdp.PacketCounter
	DropConf    map[uint32]ebpfxdp.DropPKT
)

type Statistic struct {
	CounterStat
	DropConf
}
type EbfProgram struct {
	Collection *ebpf.Collection
	lMux       sync.Mutex
	Links      map[int]link.Link
}

func LoadCollection() (*EbfProgram, error) {
	prog := &EbfProgram{
		Links: make(map[int]link.Link),
	}
	col, err := ebpfxdp.LoadCollection()
	if err != nil {
		return nil, err
	}
	prog.Collection = col

	return prog, err
}

func (e *EbfProgram) AttachXDPToNetDevice(ndev int) error {
	link, err := link.AttachXDP(
		link.XDPOptions{
			Program:   e.Collection.Programs[ebpfxdp.ProgramName],
			Interface: ndev,
			Flags:     link.XDPGenericMode,
		})
	if err != nil {
		return err
	}

	if err := e.addNetDevToMaps(ndev); err != nil {
		return err
	}

	e.lMux.Lock()
	defer e.lMux.Unlock()
	e.Links[ndev] = link

	return nil
}

func (e *EbfProgram) addNetDevToMaps(ndev int) error {
	tmpMap := e.Collection.Maps[ebpfxdp.StatsMapName]
	if err := tmpMap.Put(uint32(ndev), ebpfxdp.PacketCounter{}); err != nil { //nolint
		return err
	}
	tmpMap = e.Collection.Maps[ebpfxdp.DropMapName]
	if err := tmpMap.Put(uint32(ndev), ebpfxdp.DropPKT{}); err != nil { //nolint
		return err
	}

	return nil
}

func (e *EbfProgram) removeNetDevFromMaps(ndev int) error {
	tmpMap := e.Collection.Maps[ebpfxdp.StatsMapName]
	if err := tmpMap.Delete(uint32(ndev)); err != nil { //nolint
		return err
	}
	tmpMap = e.Collection.Maps[ebpfxdp.DropMapName]
	if err := tmpMap.Delete(uint32(ndev)); err != nil { //nolint
		return err
	}

	return nil
}

func (e *EbfProgram) DetachXDP(ndev int) error {
	if err := e.removeNetDevFromMaps(ndev); err != nil {
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
	e.removeNetDevFromMaps(ndev) //nolint
	xdpLink, exist := e.Links[ndev]
	if exist {
		xdpLink.Close()
	}
	e.lMux.Lock()
	defer e.lMux.Unlock()
	delete(e.Links, ndev)
}

func (e *EbfProgram) getStatsMapConf() (CounterStat, error) {
	iter := e.GetStatsMap().Iterate()
	var key uint32
	var value ebpfxdp.PacketCounter
	result := make(CounterStat, 10)
	for iter.Next(&key, &value) {
		result[key] = value
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (e *EbfProgram) getDropMapConf() (DropConf, error) {
	iter := e.GetDropMap().Iterate()
	var key uint32
	var value ebpfxdp.DropPKT
	result := make(DropConf, 10)
	for iter.Next(&key, &value) {
		result[key] = value
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (e *EbfProgram) GetStatistic() (*Statistic, error) {
	result := Statistic{}
	stats, err := e.getStatsMapConf()
	if err != nil {
		return nil, err
	}
	result.CounterStat = stats
	dropConf, err := e.getDropMapConf()
	if err != nil {
		return nil, err
	}
	result.DropConf = dropConf

	return &result, nil
}

func (e *EbfProgram) GetStatsMap() *ebpf.Map {
	return e.Collection.Maps[ebpfxdp.StatsMapName]
}

func (e *EbfProgram) GetDropMap() *ebpf.Map {
	return e.Collection.Maps[ebpfxdp.DropMapName]
}

func (e *EbfProgram) Close() {
	e.lMux.Lock()
	for _, ln := range e.Links {
		ln.Close()
	}
	e.lMux.Unlock()

	e.Collection.Close()
}

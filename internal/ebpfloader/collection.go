package ebpfloader

import (
	"bytes"

	"github.com/cilium/ebpf"
	"github.com/mythvcode/storm-control/ebpfxdp"
)

const (
	ProgramName  = "storm_control"
	StatsMapName = "intf_stats"
	DropMapName  = "drop_intf"
)

type (
	CounterStat map[uint32]PacketCounter
	DropConf    map[uint32]DropPKT
)

type Statistic struct {
	CounterStat
	DropConf
}
type TrafInfo struct {
	Passed  uint64
	Dropped uint64
}

type PacketCounter struct {
	Broadcast  TrafInfo
	IPv4MCast  TrafInfo
	IPv6MCast  TrafInfo
	OtherMcast TrafInfo
}

type DropPKT struct {
	Broadcast uint8
	IPv4MCast uint8
	IPv6MCast uint8
	Multicast uint8
}

type collection struct {
	*ebpf.Collection
}

func cpuCount() int {
	res, err := ebpf.PossibleCPU()
	if err != nil {
		return 0
	}

	return res
}

func getSpecs() (specs *ebpf.CollectionSpec, err error) {
	specs, err = ebpf.LoadCollectionSpecFromReader(
		bytes.NewReader(ebpfxdp.KernelProgramBytes),
	)

	return
}

func loadCollection() (*collection, error) {
	specs, err := getSpecs()
	if err != nil {
		return nil, err
	}

	statcollection := new(collection)
	statcollection.Collection, err = ebpf.NewCollection(specs)

	return statcollection, err
}

func (c *collection) getStatsMap() *ebpf.Map {
	return c.Collection.Maps[StatsMapName]
}

func (c *collection) getDropMap() *ebpf.Map {
	return c.Collection.Maps[DropMapName]
}

func (c *collection) getProgram() *ebpf.Program {
	return c.Collection.Programs[ProgramName]
}

func (c *collection) getStatsMapValues() (CounterStat, error) {
	iter := c.getStatsMap().Iterate()
	var key uint32
	perCPUValue := make([]PacketCounter, 0, cpuCount())
	result := make(CounterStat, cpuCount())
	for iter.Next(&key, &perCPUValue) {
		result[key] = mergeStat(perCPUValue)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	perCPUValue = make([]PacketCounter, 0, cpuCount())

	return result, nil
}

func (c *collection) getDropMapValues() (DropConf, error) {
	iter := c.getDropMap().Iterate()
	var key uint32
	var value DropPKT
	result := make(DropConf, cpuCount())
	for iter.Next(&key, &value) {
		result[key] = value
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *collection) putStatValue(key uint32) error {
	statMap := c.getStatsMap()
	insert := make([]PacketCounter, 0)
	if err := statMap.Put(key, insert); err != nil {
		return err
	}

	return nil
}

func (c *collection) putDropValue(key uint32, conf DropPKT) error {
	if err := c.getDropMap().Put(key, conf); err != nil {
		return err
	}

	return nil
}

func (c *collection) updateDropValue(key uint32, conf DropPKT) error {
	if err := c.getDropMap().Update(key, conf, ebpf.UpdateExist); err != nil {
		return err
	}

	return nil
}

func (c *collection) deleteStatValue(key uint32) error {
	return c.getStatsMap().Delete(key)
}

func (c *collection) deleteDropValue(key uint32) error {
	return c.getDropMap().Delete(key)
}

func (c *collection) getStatistic() (Statistic, error) {
	result := Statistic{}
	stats, err := c.getStatsMapValues()
	if err != nil {
		return Statistic{}, err
	}
	result.CounterStat = stats
	dropConf, err := c.getDropMapValues()
	if err != nil {
		return Statistic{}, err
	}
	result.DropConf = dropConf

	return result, nil
}

func (c *collection) lookupStatValue(key uint32) (PacketCounter, error) {
	perCPUResult := make([]PacketCounter, 0)
	if err := c.getStatsMap().Lookup(key, &perCPUResult); err != nil {
		return PacketCounter{}, err
	}

	return mergeStat(perCPUResult), nil
}

func (c *collection) lookupDropValue(key uint32) (DropPKT, error) {
	res := DropPKT{}
	if err := c.getDropMap().Lookup(key, &res); err != nil {
		return DropPKT{}, err
	}

	return res, nil
}

func mergeStat(resSlice []PacketCounter) PacketCounter {
	result := PacketCounter{}
	for _, resValue := range resSlice {
		result.Broadcast.Dropped += resValue.Broadcast.Dropped
		result.Broadcast.Passed += resValue.Broadcast.Passed

		result.IPv4MCast.Dropped += resValue.IPv4MCast.Dropped
		result.IPv4MCast.Passed += resValue.IPv4MCast.Passed

		result.IPv6MCast.Dropped += resValue.IPv6MCast.Dropped
		result.IPv6MCast.Passed += resValue.IPv6MCast.Passed

		result.OtherMcast.Dropped += resValue.OtherMcast.Dropped
		result.OtherMcast.Passed += resValue.OtherMcast.Passed
	}

	return result
}

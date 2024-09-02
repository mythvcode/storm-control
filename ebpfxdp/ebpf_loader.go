package ebpfxdp

import (
	"bytes"
	_ "embed"

	"github.com/cilium/ebpf"
)

const (
	ProgramName  = "storm_control"
	StatsMapName = "intf_stats"
	DropMapName  = "drop_intf"
)

type PacketCounter struct {
	Broadcast  TrafInfo
	IPv4MCast  TrafInfo
	IPv6MCast  TrafInfo
	OtherMcast TrafInfo
}

type TrafInfo struct {
	Passed  uint64
	Dropped uint64
}

type DropPKT struct {
	Broadcast uint8
	IPv4MCast uint8
	IPv6MCast uint8
	Multicast uint8
}

//go:embed xdp_kernel.o
var KernelProgramBytes []byte

func getSpecs() (specs *ebpf.CollectionSpec, err error) {
	specs, err = ebpf.LoadCollectionSpecFromReader(
		bytes.NewReader(KernelProgramBytes),
	)

	return
}

func LoadCollection() (*ebpf.Collection, error) {
	specs, err := getSpecs()
	if err != nil {
		return nil, err
	}

	return ebpf.NewCollection(specs)
}

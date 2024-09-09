package exporter

import (
	"testing"

	"github.com/mythvcode/storm-control/ebpfxdp"
	"github.com/mythvcode/storm-control/internal/ebpfloader"
)

const collectorTestZeroValues = `
# HELP storm_control_broadcast_dropped_packets Counter dropped broadcast packets by interface
# TYPE storm_control_broadcast_dropped_packets counter
storm_control_broadcast_dropped_packets{interface_index="5653",interface_name="tap72cdd785-3a"} 0
# HELP storm_control_broadcast_passed_packets Counter passed broadcast packets by interface
# TYPE storm_control_broadcast_passed_packets counter
storm_control_broadcast_passed_packets{interface_index="5653",interface_name="tap72cdd785-3a"} 0
# HELP storm_control_list_attached_interfaces List of attached interfaces
# TYPE storm_control_list_attached_interfaces gauge
storm_control_list_attached_interfaces{interface_index="5653",interface_name="tap72cdd785-3a"} 1
# HELP storm_control_multicast_dropped_packets_by_type Dropped multicast packets for interface by traffic type
# TYPE storm_control_multicast_dropped_packets_by_type counter
storm_control_multicast_dropped_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv4_multicast"} 0
storm_control_multicast_dropped_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv6_multicast"} 0
storm_control_multicast_dropped_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="other_multicast"} 0
# HELP storm_control_multicast_dropped_packets_total Total dropped multicast packets for interface
# TYPE storm_control_multicast_dropped_packets_total counter
storm_control_multicast_dropped_packets_total{interface_index="5653",interface_name="tap72cdd785-3a"} 0
# HELP storm_control_multicast_passed_packets_by_type Passed multicast packets for interface by traffic type
# TYPE storm_control_multicast_passed_packets_by_type counter
storm_control_multicast_passed_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv4_multicast"} 0
storm_control_multicast_passed_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv6_multicast"} 0
storm_control_multicast_passed_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="other_multicast"} 0
# HELP storm_control_multicast_passed_packets_total Total passed multicast packets for interface
# TYPE storm_control_multicast_passed_packets_total counter
storm_control_multicast_passed_packets_total{interface_index="5653",interface_name="tap72cdd785-3a"} 0
# HELP storm_control_traffic_blocked_status Status of blocked config for specific type of packets (0 unblocked, 1 blocked)
# TYPE storm_control_traffic_blocked_status gauge
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="broadcast"} 0
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv4_multicast"} 0
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv6_multicast"} 0
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="other_multicast"} 0
`

const collectorTestValues = `
# HELP storm_control_broadcast_dropped_packets Counter dropped broadcast packets by interface
# TYPE storm_control_broadcast_dropped_packets counter
storm_control_broadcast_dropped_packets{interface_index="5653",interface_name="tap72cdd785-3a"} 50
# HELP storm_control_broadcast_passed_packets Counter passed broadcast packets by interface
# TYPE storm_control_broadcast_passed_packets counter
storm_control_broadcast_passed_packets{interface_index="5653",interface_name="tap72cdd785-3a"} 100
# HELP storm_control_list_attached_interfaces List of attached interfaces
# TYPE storm_control_list_attached_interfaces gauge
storm_control_list_attached_interfaces{interface_index="5653",interface_name="tap72cdd785-3a"} 1
# HELP storm_control_multicast_dropped_packets_by_type Dropped multicast packets for interface by traffic type
# TYPE storm_control_multicast_dropped_packets_by_type counter
storm_control_multicast_dropped_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv4_multicast"} 1
storm_control_multicast_dropped_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv6_multicast"} 61
storm_control_multicast_dropped_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="other_multicast"} 53
# HELP storm_control_multicast_dropped_packets_total Total dropped multicast packets for interface
# TYPE storm_control_multicast_dropped_packets_total counter
storm_control_multicast_dropped_packets_total{interface_index="5653",interface_name="tap72cdd785-3a"} 115
# HELP storm_control_multicast_passed_packets_by_type Passed multicast packets for interface by traffic type
# TYPE storm_control_multicast_passed_packets_by_type counter
storm_control_multicast_passed_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv4_multicast"} 10
storm_control_multicast_passed_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv6_multicast"} 60
storm_control_multicast_passed_packets_by_type{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="other_multicast"} 55
# HELP storm_control_multicast_passed_packets_total Total passed multicast packets for interface
# TYPE storm_control_multicast_passed_packets_total counter
storm_control_multicast_passed_packets_total{interface_index="5653",interface_name="tap72cdd785-3a"} 125
# HELP storm_control_traffic_blocked_status Status of blocked config for specific type of packets (0 unblocked, 1 blocked)
# TYPE storm_control_traffic_blocked_status gauge
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="broadcast"} 1
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv4_multicast"} 0
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="ipv6_multicast"} 1
storm_control_traffic_blocked_status{interface_index="5653",interface_name="tap72cdd785-3a",traffic_type="other_multicast"} 1
`

func makeZeroTestValues(t *testing.T) (string, *ebpfloader.Statistic) {
	t.Helper()
	result := ebpfloader.Statistic{}
	result.CounterStat = ebpfloader.CounterStat{
		5653: ebpfxdp.PacketCounter{},
	}
	result.DropConf = ebpfloader.DropConf{
		5653: ebpfxdp.DropPKT{},
	}

	return collectorTestZeroValues, &result
}

func makeTestValues(t *testing.T) (string, *ebpfloader.Statistic) {
	t.Helper()
	result := ebpfloader.Statistic{}
	result.CounterStat = ebpfloader.CounterStat{
		5653: ebpfxdp.PacketCounter{
			Broadcast: ebpfxdp.TrafInfo{
				Passed:  100,
				Dropped: 50,
			},
			IPv4MCast: ebpfxdp.TrafInfo{
				Passed:  10,
				Dropped: 1,
			},
			IPv6MCast: ebpfxdp.TrafInfo{
				Passed:  60,
				Dropped: 61,
			},
			OtherMcast: ebpfxdp.TrafInfo{
				Passed:  55,
				Dropped: 53,
			},
		},
	}
	result.DropConf = ebpfloader.DropConf{
		5653: ebpfxdp.DropPKT{
			Broadcast: 1,
			IPv4MCast: 0,
			IPv6MCast: 1,
			Multicast: 1,
		},
	}

	return collectorTestValues, &result
}

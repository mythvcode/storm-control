package exporter

import (
	"log/slog"
	"net"
	"strconv"

	"github.com/mythvcode/storm-control/ebpfxdp"
	"github.com/mythvcode/storm-control/internal/ebpfloader"
	"github.com/mythvcode/storm-control/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
)

var listInterfaces = net.Interfaces

const (
	metricsNamespace    = "storm_control"
	interfaceIndexLabel = "interface_index"
	interfaceNameLabel  = "interface_name"

	trafficTypeLabel   = "traffic_type"
	broadcastType      = "broadcast"
	ipv4MulticastType  = "ipv4_multicast"
	ipv6MulticastType  = "ipv6_multicast"
	otherMulticastType = "other_multicast"
)

type StormControlCollector struct {
	statsLoader             StatsLoader
	log                     *logger.Logger
	BroadcastPassedPackets  *prometheus.CounterVec
	BroadcastDroppedPackets *prometheus.CounterVec

	MulticastPassedPacketsTotal  *prometheus.CounterVec
	MulticastDroppedPacketsTotal *prometheus.CounterVec

	MulticastPassedPacketsByType  *prometheus.CounterVec
	MulticastDroppedPacketsByType *prometheus.CounterVec

	TrafficBlockedByInterface *prometheus.GaugeVec

	AttachedLinks *prometheus.GaugeVec
}

func findInterface(netDevList []net.Interface, index uint32) *net.Interface {
	for _, netDev := range netDevList {
		if netDev.Index == int(index) {
			return &netDev
		}
	}

	return nil
}

func newStormControlCollector(statsLoader StatsLoader) *StormControlCollector {
	collector := StormControlCollector{
		statsLoader: statsLoader,
		log:         logger.GetLogger().With(slog.String(logger.Component, "prometheus-collector")),

		BroadcastPassedPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "broadcast_passed_packets",
				Help:      "Counter passed broadcast packets by interface",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel},
		),
		BroadcastDroppedPackets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "broadcast_dropped_packets",
				Help:      "Counter dropped broadcast packets by interface",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel},
		),
		MulticastPassedPacketsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "multicast_passed_packets_total",
				Help:      "Total passed multicast packets for interface",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel},
		),
		MulticastDroppedPacketsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "multicast_dropped_packets_total",
				Help:      "Total dropped multicast packets for interface",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel},
		),
		MulticastPassedPacketsByType: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "multicast_passed_packets_by_type",
				Help:      "Passed multicast packets for interface by traffic type",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel, trafficTypeLabel},
		),
		MulticastDroppedPacketsByType: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Name:      "multicast_dropped_packets_by_type",
				Help:      "Dropped multicast packets for interface by traffic type",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel, trafficTypeLabel},
		),
		TrafficBlockedByInterface: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: metricsNamespace,
				Name:      "traffic_blocked_status",
				Help:      "Status of blocked config for specific type of packets (0 unblocked, 1 blocked)",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel, trafficTypeLabel},
		),
		AttachedLinks: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: metricsNamespace,
				Name:      "list_attached_interfaces",
				Help:      "List of attached interfaces",
			},
			[]string{interfaceIndexLabel, interfaceNameLabel},
		),
	}

	return &collector
}

func (s *StormControlCollector) Initialized() bool {
	return !(s.statsLoader == nil && s.log != nil)
}
func (s *StormControlCollector) Name() string { return "storm-control-exporter" }

func (s *StormControlCollector) collectorList() []prometheus.Collector {
	return []prometheus.Collector{
		s.BroadcastPassedPackets,
		s.MulticastPassedPacketsByType,
		s.MulticastPassedPacketsTotal,

		s.BroadcastDroppedPackets,
		s.MulticastDroppedPacketsByType,
		s.MulticastDroppedPacketsTotal,

		s.TrafficBlockedByInterface,
		s.AttachedLinks,
	}
}

func (s *StormControlCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range s.collectorList() {
		metric.Describe(ch)
	}
}

func (s *StormControlCollector) calcPassedStatsForNetDev(stats *ebpfxdp.PacketCounter, netDev *net.Interface) {
	s.BroadcastPassedPackets.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
		},
	).Add(float64(stats.Broadcast.Passed))

	s.MulticastPassedPacketsByType.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
			trafficTypeLabel:    ipv4MulticastType,
		},
	).Add(float64(stats.IPv4MCast.Passed))

	s.MulticastPassedPacketsByType.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
			trafficTypeLabel:    ipv6MulticastType,
		},
	).Add(float64(stats.IPv6MCast.Passed))

	s.MulticastPassedPacketsByType.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
			trafficTypeLabel:    otherMulticastType,
		},
	).Add(float64(stats.OtherMcast.Passed))

	s.MulticastPassedPacketsTotal.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
		},
	).Add(float64(stats.IPv4MCast.Passed + stats.IPv6MCast.Passed + stats.OtherMcast.Passed))
}

func (s *StormControlCollector) calcDroppedStatsForNetDev(stats *ebpfxdp.PacketCounter, netDev *net.Interface) {
	s.BroadcastDroppedPackets.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
		},
	).Add(float64(stats.Broadcast.Dropped))

	s.MulticastDroppedPacketsByType.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
			trafficTypeLabel:    ipv4MulticastType,
		},
	).Add(float64(stats.IPv4MCast.Dropped))

	s.MulticastDroppedPacketsByType.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
			trafficTypeLabel:    ipv6MulticastType,
		},
	).Add(float64(stats.IPv6MCast.Dropped))

	s.MulticastDroppedPacketsByType.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
			trafficTypeLabel:    otherMulticastType,
		},
	).Add(float64(stats.OtherMcast.Dropped))

	s.MulticastDroppedPacketsTotal.With(
		prometheus.Labels{
			interfaceIndexLabel: strconv.Itoa(netDev.Index),
			interfaceNameLabel:  netDev.Name,
		},
	).Add(float64(stats.IPv4MCast.Dropped + stats.IPv6MCast.Dropped + stats.OtherMcast.Dropped))
}

func (s *StormControlCollector) collectStats(stats *ebpfloader.Statistic, netDevList []net.Interface) {
	for index, stats := range stats.CounterStat {
		if netDev := findInterface(netDevList, index); netDev != nil {
			s.calcPassedStatsForNetDev(&stats, netDev)
			s.calcDroppedStatsForNetDev(&stats, netDev)
		}
	}
}

func (s *StormControlCollector) collectDropConfig(stats *ebpfloader.Statistic, netDevList []net.Interface) {
	for index, stats := range stats.DropConf {
		if netDev := findInterface(netDevList, index); netDev != nil {
			s.TrafficBlockedByInterface.With(
				prometheus.Labels{
					interfaceIndexLabel: strconv.Itoa(int(index)),
					interfaceNameLabel:  netDev.Name,
					trafficTypeLabel:    broadcastType,
				},
			).Set(float64(stats.Broadcast))

			s.TrafficBlockedByInterface.With(
				prometheus.Labels{
					interfaceIndexLabel: strconv.Itoa(int(index)),
					interfaceNameLabel:  netDev.Name,
					trafficTypeLabel:    ipv4MulticastType,
				},
			).Set(float64(stats.IPv4MCast))

			s.TrafficBlockedByInterface.With(
				prometheus.Labels{
					interfaceIndexLabel: strconv.Itoa(int(index)),
					interfaceNameLabel:  netDev.Name,
					trafficTypeLabel:    ipv6MulticastType,
				},
			).Set(float64(stats.IPv6MCast))

			s.TrafficBlockedByInterface.With(
				prometheus.Labels{
					interfaceIndexLabel: strconv.Itoa(int(index)),
					interfaceNameLabel:  netDev.Name,
					trafficTypeLabel:    otherMulticastType,
				},
			).Set(float64(stats.Multicast))
		}
	}
}

func (s *StormControlCollector) collectAttachedInterfaces(stats *ebpfloader.Statistic, netDevList []net.Interface) {
	for index := range stats.CounterStat {
		if netDev := findInterface(netDevList, index); netDev != nil {
			s.AttachedLinks.With(
				prometheus.Labels{
					interfaceIndexLabel: strconv.Itoa(int(index)),
					interfaceNameLabel:  netDev.Name,
				},
			).Set(1)
		}
	}
}

// Collect sends all the collected metrics to the provided Prometheus channel.
// It requires the caller to handle synchronization.
func (s *StormControlCollector) Collect(metricChan chan<- prometheus.Metric) {
	// Reset current statistic.
	s.BroadcastPassedPackets.Reset()
	s.BroadcastDroppedPackets.Reset()

	s.MulticastPassedPacketsByType.Reset()
	s.MulticastPassedPacketsTotal.Reset()

	s.MulticastDroppedPacketsByType.Reset()
	s.MulticastDroppedPacketsTotal.Reset()

	s.TrafficBlockedByInterface.Reset()
	s.AttachedLinks.Reset()

	stats, err := s.statsLoader.GetStatistic()
	if err != nil {
		s.log.Errorf("Error collect eBPF statistics: %s", err.Error())

		return
	}
	netDevList, err := listInterfaces()
	if err != nil {
		s.log.Errorf("Error get list of network devices: %s", err.Error())

		return
	}

	s.collectStats(stats, netDevList)
	s.collectDropConfig(stats, netDevList)
	s.collectAttachedInterfaces(stats, netDevList)

	for _, metric := range s.collectorList() {
		metric.Collect(metricChan)
	}
}

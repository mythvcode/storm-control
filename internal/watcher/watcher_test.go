package watcher

import (
	"context"
	"log/slog"
	"net"
	"regexp"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/mythvcode/storm-control/internal/config"
	"github.com/mythvcode/storm-control/internal/logger"
	"github.com/mythvcode/storm-control/internal/watcher/mocks"
	"github.com/stretchr/testify/require"
)

type discardHandler struct {
	slog.JSONHandler
}

func (discardHandler) Enabled(context.Context, slog.Level) bool {
	return false
}

func (discardHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (h discardHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h discardHandler) WithGroup(string) slog.Handler {
	return h
}

func init() {
	slog.SetDefault(slog.New(&discardHandler{}))
	setListInterfaceFunc()
}

func setListInterfaceFunc() {
	listInterfaces = func() ([]net.Interface, error) {
		return []net.Interface{
			{
				Index: 123,
				Name:  "tap123",
			},
			{
				Index: 1,
				Name:  "tap1",
			},
			{
				Index: 5,
				Name:  "tap5",
			},
			{
				Index: 100,
				Name:  "notTap",
			},
		}, nil
	}
}

func makeTestWatcher(t *testing.T) (*Watcher, *mocks.MockeBPFProg) {
	t.Helper()
	netDevRegexp := "^tap."
	ebpMock := mocks.NewMockeBPFProg(t)

	return &Watcher{
		devWatcherMap: make(map[int]*netDevWatcher),
		ebpfProg:      ebpMock,
		config:        config.WatcherConfig{DevRegEx: netDevRegexp, BlockEnabled: false},
		netDevReg:     regexp.MustCompile(netDevRegexp),
		log:           logger.GetLogger(),
	}, ebpMock
}

func TestFindStaticNetDevices(t *testing.T) {
	watcher, _ := makeTestWatcher(t)
	watcher.config.StaticDevList = []string{"tap123", "tap5"}
	allNetDevices := []net.Interface{
		{
			Index: 123,
			Name:  "tap123",
		},
		{
			Index: 1,
			Name:  "tap1",
		},
		{
			Index: 5,
			Name:  "tap5",
		},
	}
	res, err := watcher.getNetDevicesForAttach()
	require.NoError(t, err)
	require.Equal(t, []net.Interface{allNetDevices[0], allNetDevices[2]}, res)
}

func TestFindDevices(t *testing.T) {
	watcher, _ := makeTestWatcher(t)
	res, err := watcher.getNetDevicesForAttach()
	require.NoError(t, err)
	expectedDevs := []net.Interface{
		{
			Index: 123,
			Name:  "tap123",
		},
		{
			Index: 1,
			Name:  "tap1",
		},
		{
			Index: 5,
			Name:  "tap5",
		},
	}

	require.Len(t, res, 3)
	for _, dev := range expectedDevs {
		require.Contains(t, res, dev)
	}
}

func TestAttachProgram(t *testing.T) {
	watcher, ebpfMock := makeTestWatcher(t)
	ebpfMock.EXPECT().AttachXDPToNetDevice(1).Return(nil)
	ebpfMock.EXPECT().AttachXDPToNetDevice(123).Return(nil)
	ebpfMock.EXPECT().AttachXDPToNetDevice(5).Return(nil)
	ebpfMock.EXPECT().GetStatsMap().Return(&ebpf.Map{})
	ebpfMock.EXPECT().GetDropMap().Return(&ebpf.Map{})
	watcher.findAndAttachNetDev()
	ebpfMock.AssertNotCalled(t, "AttachXDPToNetDevice", 100)
	ebpfMock.AssertNumberOfCalls(t, "AttachXDPToNetDevice", 3)
	ebpfMock.AssertNumberOfCalls(t, "GetStatsMap", 3)
	ebpfMock.AssertNumberOfCalls(t, "GetDropMap", 3)
}

func TestDetachProg(t *testing.T) {
	watcher, ebpfMock := makeTestWatcher(t)
	ebpfMock.EXPECT().AttachXDPToNetDevice(1).Return(nil)
	ebpfMock.EXPECT().AttachXDPToNetDevice(123).Return(nil)
	ebpfMock.EXPECT().AttachXDPToNetDevice(5).Return(nil)
	ebpfMock.EXPECT().GetStatsMap().Return(&ebpf.Map{})
	ebpfMock.EXPECT().GetDropMap().Return(&ebpf.Map{})
	watcher.findAndAttachNetDev()
	listInterfaces = func() ([]net.Interface, error) {
		return []net.Interface{
			{
				Index: 1,
				Name:  "tap1",
			},
		}, nil
	}
	defer setListInterfaceFunc()
	ebpfMock.EXPECT().ForceDetachXDP(123)
	ebpfMock.EXPECT().ForceDetachXDP(5)
	watcher.cleanNetDev()
}

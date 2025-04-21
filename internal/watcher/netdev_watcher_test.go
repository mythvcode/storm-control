package watcher

import (
	"errors"
	"testing"

	"github.com/mythvcode/storm-control/internal/ebpfloader"
	"github.com/mythvcode/storm-control/internal/watcher/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createWatcher(t *testing.T) *netDevWatcher {
	t.Helper()

	return newNetDevWatcher(1, "test_name", 10, 0, mocks.NewMockeBPFProg(t))
}

func TestAcquireBlockState(t *testing.T) {
	tCases := []struct {
		trafType int
		statsus  bool
	}{
		{
			trafType: 0,
			statsus:  false,
		},
		{
			trafType: broadcastType,
			statsus:  true,
		},
		{
			trafType: ipv4McastType,
			statsus:  true,
		},
		{
			trafType: ipv6McastType,
			statsus:  true,
		},
		{
			trafType: otherType,
			statsus:  true,
		},
		{
			trafType: 100,
			statsus:  false,
		},
	}

	for _, tCase := range tCases {
		watcher := createWatcher(t)
		if tCase.statsus == true {
			require.True(t, watcher.acquireBlockState(tCase.trafType))
		} else {
			require.False(t, watcher.acquireBlockState(tCase.trafType))
		}
	}
}

func TestAcquireBlockStateSecondTime(t *testing.T) {
	watcher := createWatcher(t)
	require.True(t, watcher.acquireBlockState(broadcastType))
	require.False(t, watcher.acquireBlockState(broadcastType))
}

func TestReleaseBlockState(t *testing.T) {
	watcher := createWatcher(t)
	require.True(t, watcher.acquireBlockState(broadcastType))
	require.True(t, watcher.acquireBlockState(ipv4McastType))
	require.True(t, watcher.acquireBlockState(ipv6McastType))
	require.True(t, watcher.acquireBlockState(otherType))
	require.True(t, watcher.dropState.brDropped.Load())
	require.True(t, watcher.dropState.ipv4McastDropped.Load())
	require.True(t, watcher.dropState.ipv6McastDropped.Load())
	require.True(t, watcher.dropState.other.Load())

	require.False(t, watcher.acquireBlockState(broadcastType))
	require.False(t, watcher.acquireBlockState(ipv4McastType))
	require.False(t, watcher.acquireBlockState(ipv6McastType))
	require.False(t, watcher.acquireBlockState(otherType))
}

func TestDevInfo(t *testing.T) {
	ebpfProg := mocks.NewMockeBPFProg(t)

	watchr := newNetDevWatcher(1, "test_name", 10, 0, ebpfProg)
	res := watchr.devInfo()
	require.Equal(t, "test_name (1)", res)
}

func TestUpdateDropConf(t *testing.T) {
	ebpfProg := mocks.NewMockeBPFProg(t)
	ebpfProg.EXPECT().GetDevDropCfg(mock.Anything).Return(ebpfloader.DropPKT{}, nil)
	ebpfProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{Broadcast: 1}).Return(nil)
	ebpfProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{IPv4MCast: 1}).Return(nil)
	ebpfProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{IPv6MCast: 1, Multicast: 1}).Return(nil)
	watchr := newNetDevWatcher(1, "test_name", 10, 0, ebpfProg)
	require.NoError(t, watchr.updateDropMap(updateDropConfig{br: blockAction}))
	require.NoError(t, watchr.updateDropMap(updateDropConfig{ipv4: blockAction}))
	require.NoError(t, watchr.updateDropMap(updateDropConfig{ipv6: blockAction, other: blockAction}))
	// unblock
	ebpfProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{Broadcast: 0, IPv4MCast: 0}).Return(nil)
	ebpfProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{IPv6MCast: 1, Multicast: 0}).Return(nil)
	require.NoError(t, watchr.updateDropMap(updateDropConfig{br: unblockAction, ipv4: unblockAction}))
	require.NoError(t, watchr.updateDropMap(updateDropConfig{ipv6: blockAction, other: unblockAction}))
}

func TestCalculateStats(t *testing.T) {
	watcher := createWatcher(t)
	calcFunc := watcher.getCalculateStatsFuc()
	blockConf := calcFunc(ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Passed: 100}})
	require.Equal(t, updateDropConfig{br: blockAction}, blockConf)
	blockConf = calcFunc(ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Passed: 200}, IPv4MCast: ebpfloader.TrafInfo{Passed: 100}})
	require.Equal(t, updateDropConfig{br: blockAction, ipv4: blockAction}, blockConf)
	blockConf = calcFunc(ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Passed: 200}, IPv4MCast: ebpfloader.TrafInfo{Passed: 100}})
	require.Equal(t, updateDropConfig{}, blockConf)
}

func TestCheckUnblockError(t *testing.T) {
	ebpfProg := mocks.NewMockeBPFProg(t)
	watcher := newNetDevWatcher(1, "test_name", 10, 0, ebpfProg)
	unblock, err := watcher.checkAndUnblock(
		&ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Dropped: 100}},
		&ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Dropped: 200}},
		broadcastType,
	)
	require.NoError(t, err)
	require.False(t, unblock)
	ebpfProg.EXPECT().GetDevDropCfg(1).Return(ebpfloader.DropPKT{}, nil)
	ebpfProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{Broadcast: 0}).Return(errors.New("error map drop config"))
	unblock, err = watcher.checkAndUnblock(
		&ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Dropped: 100}},
		&ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Dropped: 101}},
		broadcastType,
	)
	require.False(t, unblock)
	require.Error(t, err)
	require.Equal(t, "error map drop config", err.Error())
}

func TestCheckUnblock(t *testing.T) {
	tCases := []struct {
		initMock     func(dropMock *mocks.MockeBPFProg)
		unblockCheck func(watcher *netDevWatcher) (bool, error)
	}{
		{
			initMock: func(eBPFProg *mocks.MockeBPFProg) {
				eBPFProg.EXPECT().GetDevDropCfg(1).Return(ebpfloader.DropPKT{}, nil)
				eBPFProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{Broadcast: 0}).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Dropped: 100}},
					&ebpfloader.PacketCounter{Broadcast: ebpfloader.TrafInfo{Dropped: 101}},
					broadcastType,
				)
			},
		},
		{
			initMock: func(eBPFProg *mocks.MockeBPFProg) {
				eBPFProg.EXPECT().GetDevDropCfg(1).Return(ebpfloader.DropPKT{}, nil)
				eBPFProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{IPv4MCast: 0}).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfloader.PacketCounter{IPv4MCast: ebpfloader.TrafInfo{Dropped: 100}},
					&ebpfloader.PacketCounter{IPv4MCast: ebpfloader.TrafInfo{Dropped: 101}},
					ipv4McastType,
				)
			},
		},
		{
			initMock: func(eBPFProg *mocks.MockeBPFProg) {
				eBPFProg.EXPECT().GetDevDropCfg(1).Return(ebpfloader.DropPKT{}, nil)
				eBPFProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{IPv6MCast: 0}).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfloader.PacketCounter{IPv6MCast: ebpfloader.TrafInfo{Dropped: 100}},
					&ebpfloader.PacketCounter{IPv6MCast: ebpfloader.TrafInfo{Dropped: 101}},
					ipv6McastType,
				)
			},
		},
		{
			initMock: func(eBPFProg *mocks.MockeBPFProg) {
				eBPFProg.EXPECT().GetDevDropCfg(1).Return(ebpfloader.DropPKT{}, nil)
				eBPFProg.EXPECT().UpdateDevDropCfg(1, ebpfloader.DropPKT{Multicast: 0}).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfloader.PacketCounter{OtherMcast: ebpfloader.TrafInfo{Dropped: 100}},
					&ebpfloader.PacketCounter{OtherMcast: ebpfloader.TrafInfo{Dropped: 101}},
					otherType,
				)
			},
		},
	}
	for _, tCase := range tCases {
		ebpfProg := mocks.NewMockeBPFProg(t)
		watcher := newNetDevWatcher(1, "test_name", 10, 0, ebpfProg)
		tCase.initMock(ebpfProg)
		unblocked, err := tCase.unblockCheck(watcher)
		require.NoError(t, err)
		require.True(t, unblocked)
	}
}

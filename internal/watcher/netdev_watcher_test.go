package watcher

import (
	"errors"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/mythvcode/storm-control/ebpfxdp"
	"github.com/mythvcode/storm-control/internal/watcher/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func createWatcher(t *testing.T) *netDevWatcher {
	t.Helper()

	return newNetDevWatcher(1, "test_name", 10, 0, &ebpf.Map{}, &ebpf.Map{})
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
	statsMock := mocks.NewMockEBPFMap(t)
	dropMock := mocks.NewMockEBPFMap(t)
	watchr := newNetDevWatcher(1, "test_name", 10, 0, statsMock, dropMock)
	res := watchr.devInfo()
	require.Equal(t, "test_name (1)", res)
}

func TestUpdateDropMap(t *testing.T) {
	statsMock := mocks.NewMockEBPFMap(t)
	dropMock := mocks.NewMockEBPFMap(t)
	dropMock.EXPECT().Lookup(mock.Anything, mock.Anything).Return(nil)
	dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{Broadcast: 1}, ebpf.UpdateExist).Return(nil)
	dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{IPv4MCast: 1}, ebpf.UpdateExist).Return(nil)
	dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{IPv6MCast: 1, Multicast: 1}, ebpf.UpdateExist).Return(nil)
	watchr := newNetDevWatcher(1, "test_name", 10, 0, statsMock, dropMock)
	require.NoError(t, watchr.updateDropMap(updateDropConfig{br: blockAction}))
	require.NoError(t, watchr.updateDropMap(updateDropConfig{ipv4: blockAction}))
	require.NoError(t, watchr.updateDropMap(updateDropConfig{ipv6: blockAction, other: blockAction}))
	// unblock
	dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{Broadcast: 0, IPv4MCast: 0}, ebpf.UpdateExist).Return(nil)
	dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{IPv6MCast: 1, Multicast: 0}, ebpf.UpdateExist).Return(nil)
	require.NoError(t, watchr.updateDropMap(updateDropConfig{br: unblockAction, ipv4: unblockAction}))
	require.NoError(t, watchr.updateDropMap(updateDropConfig{ipv6: blockAction, other: unblockAction}))
}

func TestCalculateStats(t *testing.T) {
	watcher := createWatcher(t)
	calcFunc := watcher.getCalculateStatsFuc()
	blockConf := calcFunc(ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Passed: 100}})
	require.Equal(t, updateDropConfig{br: blockAction}, blockConf)
	blockConf = calcFunc(ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Passed: 200}, IPv4MCast: ebpfxdp.TrafInfo{Passed: 100}})
	require.Equal(t, updateDropConfig{br: blockAction, ipv4: blockAction}, blockConf)
	blockConf = calcFunc(ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Passed: 200}, IPv4MCast: ebpfxdp.TrafInfo{Passed: 100}})
	require.Equal(t, updateDropConfig{}, blockConf)
}

func TestCheckUnblockError(t *testing.T) {
	statsMock := mocks.NewMockEBPFMap(t)
	dropMock := mocks.NewMockEBPFMap(t)
	watcher := newNetDevWatcher(1, "test_name", 10, 0, statsMock, dropMock)
	unblock, err := watcher.checkAndUnblock(
		&ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Dropped: 100}},
		&ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Dropped: 200}},
		broadcastType,
	)
	require.NoError(t, err)
	require.False(t, unblock)
	dropMock.EXPECT().Lookup(mock.Anything, mock.Anything).Return(nil)
	dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{Broadcast: 0}, ebpf.UpdateExist).Return(errors.New("error update map"))
	unblock, err = watcher.checkAndUnblock(
		&ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Dropped: 100}},
		&ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Dropped: 101}},
		broadcastType,
	)
	require.False(t, unblock)
	require.Error(t, err)
	require.Equal(t, "error update map", err.Error())
}

func TestCheckUnblock(t *testing.T) {
	tCases := []struct {
		initMock     func(dropMock *mocks.MockEBPFMap)
		unblockCheck func(watcher *netDevWatcher) (bool, error)
	}{
		{
			initMock: func(dropMock *mocks.MockEBPFMap) {
				dropMock.EXPECT().Lookup(mock.Anything, mock.Anything).Return(nil)
				dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{Broadcast: 0}, ebpf.UpdateExist).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Dropped: 100}},
					&ebpfxdp.PacketCounter{Broadcast: ebpfxdp.TrafInfo{Dropped: 101}},
					broadcastType,
				)
			},
		},
		{
			initMock: func(dropMock *mocks.MockEBPFMap) {
				dropMock.EXPECT().Lookup(mock.Anything, mock.Anything).Return(nil)
				dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{IPv4MCast: 0}, ebpf.UpdateExist).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfxdp.PacketCounter{IPv4MCast: ebpfxdp.TrafInfo{Dropped: 100}},
					&ebpfxdp.PacketCounter{IPv4MCast: ebpfxdp.TrafInfo{Dropped: 101}},
					ipv4McastType,
				)
			},
		},
		{
			initMock: func(dropMock *mocks.MockEBPFMap) {
				dropMock.EXPECT().Lookup(mock.Anything, mock.Anything).Return(nil)
				dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{IPv6MCast: 0}, ebpf.UpdateExist).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfxdp.PacketCounter{IPv6MCast: ebpfxdp.TrafInfo{Dropped: 100}},
					&ebpfxdp.PacketCounter{IPv6MCast: ebpfxdp.TrafInfo{Dropped: 101}},
					ipv6McastType,
				)
			},
		},
		{
			initMock: func(dropMock *mocks.MockEBPFMap) {
				dropMock.EXPECT().Lookup(mock.Anything, mock.Anything).Return(nil)
				dropMock.EXPECT().Update(uint32(1), ebpfxdp.DropPKT{Multicast: 0}, ebpf.UpdateExist).Return(nil)
			},
			unblockCheck: func(watcher *netDevWatcher) (bool, error) {
				return watcher.checkAndUnblock(
					&ebpfxdp.PacketCounter{OtherMcast: ebpfxdp.TrafInfo{Dropped: 100}},
					&ebpfxdp.PacketCounter{OtherMcast: ebpfxdp.TrafInfo{Dropped: 101}},
					otherType,
				)
			},
		},
	}
	for _, tCase := range tCases {
		statsMock := mocks.NewMockEBPFMap(t)
		dropMock := mocks.NewMockEBPFMap(t)
		watcher := newNetDevWatcher(1, "test_name", 10, 0, statsMock, dropMock)
		tCase.initMock(dropMock)
		unblocked, err := tCase.unblockCheck(watcher)
		require.NoError(t, err)
		require.True(t, unblocked)
	}
}

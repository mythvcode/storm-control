package exporter

import (
	"net"
	"strings"
	"testing"

	"github.com/mythvcode/storm-control/internal/config"
	"github.com/mythvcode/storm-control/internal/exporter/mocks"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewExporter(t *testing.T) {
	mock := mocks.NewMockStatsLoader(t)
	cfg, err := config.ReadConfig("")
	require.NoError(t, err)
	_, err = New(cfg.Exporter, mock)
	require.NoError(t, err)
}

func TestCollector(t *testing.T) {
	listInterfaces = func() ([]net.Interface, error) {
		return []net.Interface{{Index: 5653, Name: "tap72cdd785-3a"}}, nil
	}
	mock := mocks.NewMockStatsLoader(t)
	raw, stats := makeZeroTestValues(t)
	mock.EXPECT().GetStatistic().Return(stats, nil).Once()
	collector := newStormControlCollector(mock)

	err := testutil.CollectAndCompare(collector, strings.NewReader(raw))
	require.NoError(t, err)
	raw, stats = makeTestValues(t)
	mock.EXPECT().GetStatistic().Return(stats, nil).Once()

	err = testutil.CollectAndCompare(collector, strings.NewReader(raw))
	require.NoError(t, err)
}

package exporter

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mythvcode/storm-control/internal/config"
	"github.com/mythvcode/storm-control/internal/ebpfloader"
	"github.com/mythvcode/storm-control/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type APIServer struct {
	server *http.Server
	log    *logger.Logger
	config config.Exporter
}

type StatsLoader interface {
	GetStatistic() (*ebpfloader.Statistic, error)
}

func New(cfg config.Exporter, statsLoader StatsLoader) (*APIServer, error) {
	apiServer := APIServer{
		log:    logger.GetLogger().With(slog.String(logger.Component, "exporter-api-server")),
		config: cfg,
	}
	collector := newStormControlCollector(statsLoader)
	if !collector.Initialized() {
		return nil, fmt.Errorf("collector %s was not initialized", collector.Name())
	}
	if err := prometheus.Register(collector); err != nil {
		return nil, err
	}

	httpMux := http.NewServeMux()
	timeout := time.Duration(cfg.RequestTimeout) * time.Second
	address := strings.Join([]string{cfg.ServerAddress, strconv.Itoa(cfg.ServerPort)}, ":")
	apiServer.server = &http.Server{
		Addr:         address,
		Handler:      httpMux,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		IdleTimeout:  timeout,
	}
	httpMux.HandleFunc("/", apiServer.indexPage)
	if cfg.EnableRequestLogging {
		httpMux.Handle(cfg.TelemetryPath, apiServer.middlewareLogging(promhttp.Handler()))
	} else {
		httpMux.Handle(cfg.TelemetryPath, promhttp.Handler())
	}
	if !cfg.EnableRuntimeMetrics {
		prometheus.Unregister(collectors.NewGoCollector())
	}

	return &apiServer, nil
}

func (s *APIServer) middlewareLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(respwr http.ResponseWriter, req *http.Request) {
		s.log.With(
			slog.String("addr", req.RemoteAddr),
			slog.String("method", req.Method),
			slog.String("agent", req.UserAgent()),
		).Debugf("%s", req.URL.Path)

		next.ServeHTTP(respwr, req)
	})
}

// StartAPIServer starts Exporter's HTTP server.
func (s *APIServer) Start() error {
	s.log.Infof("Starting exporter API server on %s", s.server.Addr)

	return s.server.ListenAndServe()
}

func (s *APIServer) Stop() {
	s.log.Infof("Stopping exporter API server")
	if err := s.server.Close(); err != nil {
		s.log.Errorf("Error stop exporter")
	}
}

func (s *APIServer) indexPage(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte(`<html>
<head><title>eBPF Storm Control Exporter</title></head>
<body>
<h1>eBPF Storm Control Exporter</h1>
<p><a href='` + s.config.TelemetryPath + `'>Metrics</a></p>
</body>
</html>`))
	if err != nil {
		s.log.Errorf("error handling index page: %s", err)
	}
}

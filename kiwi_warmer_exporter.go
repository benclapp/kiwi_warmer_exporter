package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	logLvl = flag.String("log-level", "info",
		"Log Level")
	listenAddress = flag.String("web.listen-address", ":10015",
		"Address to listen on for HTTP requests.")
	metricsPath = flag.String("web.metrics-path", "/metrics",
		"Path to expose metrics on.")
	kwAddress = flag.String("kw.address", "",
		"[REQUIRED] The address of the Kiwi Warmer.")
	kwScrapeTimeout = flag.Duration("kw.scrape-timeout", time.Second*3,
		"The timeout for the scrape request.")

	// Provided at build time
	builtBy, commit, date, version string
)

type Exporter struct{}

func main() {
	flag.Parse()
	var logLevel slog.Level
	switch *logLvl {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(
		os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     logLevel,
		})))

	_, _, err := getDeviceInfo()
	if err != nil {
		slog.Error("Error getting Device Info", "err", err)
	}

	prometheus.MustRegister(&Exporter{})

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Kiwi Warmer Exporter</title></head>
			<body>
			<h1>Kiwi Warmer Exporter</h1>
			<p><a href=/metrics>Metrics</a></p>
			</body>
			</html>`))
	})

	slog.Info("Starting Kiwi Warmer Exporter",
		"log-level", logLevel,
		"web.listen-address", *listenAddress,
		"web.metrics-path", *metricsPath,
		"kw.address", *kwAddress,
		"kw.scrape-timeout", *kwScrapeTimeout,
	)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))

}

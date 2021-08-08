package xhttp

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thecodingmachine/gotenberg/internal/pkg/printer"
)

var (
	chrome_current_rendering = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "golang",
			Name:      "chrome_current_rendering",
			Help:      "This gauge monitors the current number of rendering of chrome",
		})
)

func StartCustomMonitoring() {
	prometheus.MustRegister(chrome_current_rendering)

	go func() {
		for {
			chrome_current_rendering.Set(float64(printer.GetChromeStatus().CurrentRendering))
			time.Sleep(time.Second)
		}
	}()
}

package metrics

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// RunPrometheus configures an OTEL MeterProvider and serves metrics on /metrics.
func RunPrometheus(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	exporter, err := otelprom.New(
		otelprom.WithRegisterer(prometheus.DefaultRegisterer),
	)
	if err != nil {
		return nil, err
	}

	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		srv := &http.Server{
			Addr:              ":2112",
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           mux,
		}

		shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
		defer shutdownCancel()
		go func() {
			<-ctx.Done()
			graceCtx, cancel := context.WithTimeout(shutdownCtx, 5*time.Second)
			defer cancel()
			if err := srv.Shutdown(graceCtx); err != nil {
				log.Printf("metrics server shutdown error: %v", err)
			}
		}()

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server error: %v", err)
		}
	}()

	return provider, nil
}

package metrics

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

const (
	instrumentationName          = "github.com/thrasher-corp/gocryptotrader/metrics"
	DefaultWebsocketReaderWindow = 30 * time.Second
)

// WebsocketReaderSnapshot captures aggregated reader timing data for logging.
type WebsocketReaderSnapshot struct {
	Window            time.Duration
	Operations        uint64
	Errors            uint64
	AverageProcessing time.Duration
	OpsPerSecond      float64
	Peak              time.Duration
	PeakCause         string
}

// WebsocketReader tracks websocket reader processing times for metrics and logging.
type WebsocketReader struct {
	meter      metric.Meter
	operations metric.Int64Counter
	errors     metric.Int64Counter
	processing metric.Float64Histogram

	mu                 sync.Mutex
	window             time.Duration
	windowStart        time.Time
	operationsInWindow uint64
	errorsInWindow     uint64
	totalProcessing    time.Duration
	peakDuration       time.Duration
	peakCause          string
}

// NewWebsocketReader creates a reader metrics tracker with an optional rolling window.
func NewWebsocketReader(window time.Duration) *WebsocketReader {
	if window <= 0 {
		window = DefaultWebsocketReaderWindow
	}

	meter := otel.Meter(instrumentationName)
	operations, err := meter.Int64Counter("websocket.reader.operations")
	errors, errErrors := meter.Int64Counter("websocket.reader.errors")
	processing, errProcessing := meter.Float64Histogram("websocket.reader.processing_duration_ms", metric.WithUnit("ms"))
	if err != nil || errErrors != nil || errProcessing != nil {
		meter = noop.NewMeterProvider().Meter(instrumentationName)
		operations, _ = meter.Int64Counter("websocket.reader.operations")
		errors, _ = meter.Int64Counter("websocket.reader.errors")
		processing, _ = meter.Float64Histogram("websocket.reader.processing_duration_ms", metric.WithUnit("ms"))
	}

	return &WebsocketReader{
		meter:      meter,
		operations: operations,
		errors:     errors,
		processing: processing,
		window:     window,
	}
}

// Record tracks a single reader operation and returns a snapshot once the window elapses.
func (w *WebsocketReader) Record(ctx context.Context, exchangeName, connectionURL string, duration time.Duration, err error, cause string) (WebsocketReaderSnapshot, bool) {
	attrs := []attribute.KeyValue{
		attribute.String("exchange", exchangeName),
		attribute.String("connection", sanitizeConnectionURL(connectionURL)),
	}

	w.operations.Add(ctx, 1, metric.WithAttributes(attrs...))
	w.processing.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
	if err != nil {
		w.errors.Add(ctx, 1, metric.WithAttributes(attrs...))
		if cause == "" {
			cause = err.Error()
		}
	}

	now := time.Now()

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.windowStart.IsZero() {
		w.windowStart = now
	}

	w.operationsInWindow++
	if err != nil {
		w.errorsInWindow++
	}
	w.totalProcessing += duration
	if duration > w.peakDuration {
		w.peakDuration = duration
		w.peakCause = cause
	}

	windowElapsed := now.Sub(w.windowStart)
	if windowElapsed < w.window {
		return WebsocketReaderSnapshot{}, false
	}

	snapshot := WebsocketReaderSnapshot{
		Window:     windowElapsed,
		Operations: w.operationsInWindow,
		Errors:     w.errorsInWindow,
		Peak:       w.peakDuration,
		PeakCause:  w.peakCause,
	}
	if snapshot.Operations > 0 {
		snapshot.AverageProcessing = time.Duration(int64(w.totalProcessing) / int64(snapshot.Operations))
		if snapshot.Window > 0 {
			snapshot.OpsPerSecond = float64(snapshot.Operations) / snapshot.Window.Seconds()
		}
	}

	w.windowStart = now
	w.operationsInWindow = 0
	w.errorsInWindow = 0
	w.totalProcessing = 0
	w.peakDuration = 0
	w.peakCause = ""

	return snapshot, true
}

func sanitizeConnectionURL(u string) string {
	baseURL, _, ok := strings.Cut(u, "?")
	if !ok {
		return u
	}
	return baseURL
}

package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerProvider holds the OpenTelemetry tracer provider and configuration
type TracerProvider struct {
	provider *sdktrace.TracerProvider
	enabled  bool
}

// InitTracer initializes OpenTelemetry tracing with OTLP HTTP exporter
// Configuration via environment variables:
//
//	OTEL_ENABLED - Enable/disable tracing (default: false)
//	OTEL_EXPORTER_OTLP_ENDPOINT - OTLP endpoint URL (default: http://localhost:4318)
//	OTEL_SERVICE_NAME - Service name for traces (default: innominatus)
//	OTEL_SERVICE_VERSION - Service version (optional)
func InitTracer(version, commit string) (*TracerProvider, error) {
	// Check if tracing is enabled
	enabled := os.Getenv("OTEL_ENABLED") == "true"
	if !enabled {
		return &TracerProvider{enabled: false}, nil
	}

	// Get OTLP endpoint (default to localhost:4318 for HTTP)
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	// Get service name
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "innominatus"
	}

	// Get service version
	serviceVersion := os.Getenv("OTEL_SERVICE_VERSION")
	if serviceVersion == "" {
		serviceVersion = version
	}

	// Create OTLP HTTP exporter
	ctx := context.Background()
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(getEndpointHost(endpoint)),
		otlptracehttp.WithInsecure(), // Use WithTLSClientConfig for production with TLS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP HTTP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create trace provider with batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		// Sample all traces in development, or use probabilistic sampling in production
		sdktrace.WithSampler(getSampler()),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator to W3C Trace Context
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		provider: tp,
		enabled:  true,
	}, nil
}

// getEndpointHost extracts host:port from URL or returns as-is
func getEndpointHost(endpoint string) string {
	// Remove http:// or https:// prefix
	if len(endpoint) > 7 && endpoint[:7] == "http://" {
		return endpoint[7:]
	}
	if len(endpoint) > 8 && endpoint[:8] == "https://" {
		return endpoint[8:]
	}
	return endpoint
}

// getSampler returns the appropriate sampler based on environment
func getSampler() sdktrace.Sampler {
	env := os.Getenv("ENV")
	sampleRate := os.Getenv("OTEL_TRACE_SAMPLE_RATE")

	if env == "production" && sampleRate != "" {
		// Use probabilistic sampling in production
		// Example: OTEL_TRACE_SAMPLE_RATE=0.1 for 10% sampling
		var rate float64
		if _, err := fmt.Sscanf(sampleRate, "%f", &rate); err == nil && rate >= 0 && rate <= 1 {
			return sdktrace.TraceIDRatioBased(rate)
		}
		// Fall through to AlwaysSample if parsing fails
	}

	// Always sample in development
	return sdktrace.AlwaysSample()
}

// Shutdown gracefully shuts down the tracer provider
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if !tp.enabled || tp.provider == nil {
		return nil
	}
	return tp.provider.Shutdown(ctx)
}

// GetTracer returns a tracer for the given component name
func (tp *TracerProvider) GetTracer(name string) trace.Tracer {
	if !tp.enabled {
		return noop.NewTracerProvider().Tracer(name)
	}
	return otel.Tracer(name)
}

// IsEnabled returns whether tracing is enabled
func (tp *TracerProvider) IsEnabled() bool {
	return tp.enabled
}

// StartSpan is a convenience function to start a new span
func StartSpan(ctx context.Context, tracerName, spanName string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName)
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(ctx context.Context, name, message string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes())
	}
}

// SetSpanStatus sets the status of the current span
func SetSpanStatus(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
	}
}

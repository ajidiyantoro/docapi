package otel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Init initializes the OpenTelemetry tracer provider with OTLP exporter.
func Init(ctx context.Context, loc *time.Location) (func(context.Context) error, error) {
	if os.Getenv("OTEL_SDK_DISABLED") == "true" {
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		logStartup(loc, false, "", "", "", "")
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(getEnv("OTEL_SERVICE_NAME", "docapi")),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	if protocol == "" {
		protocol = "grpc" // default OTLP protocol
	}

	var exporter *otlptrace.Exporter
	var expErr error

	switch protocol {
	case "grpc":
		exporter, expErr = otlptracegrpc.New(ctx)
	case "http/protobuf":
		exporter, expErr = otlptracehttp.New(ctx)
	default:
		expErr = fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}

	if expErr != nil {
		logError(loc, expErr)
		// Degrade gracefully: set noop tracer provider
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
		return func(context.Context) error { return nil }, nil
	}

	sampler := getSampler()

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	samplerType := os.Getenv("OTEL_TRACES_SAMPLER")
	if samplerType == "" {
		samplerType = "parentbased_traceidratio"
	}
	samplerArg := os.Getenv("OTEL_TRACES_SAMPLER_ARG")
	if samplerArg == "" {
		samplerArg = "1.0"
	}

	logStartup(loc, true, protocol, endpoint, samplerType, samplerArg)

	return tp.Shutdown, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getSampler() trace.Sampler {
	sampler := os.Getenv("OTEL_TRACES_SAMPLER")
	arg := os.Getenv("OTEL_TRACES_SAMPLER_ARG")

	switch sampler {
	case "always_on":
		return trace.AlwaysSample()
	case "always_off":
		return trace.NeverSample()
	case "traceidratio":
		var ratio float64 = 1.0
		fmt.Sscanf(arg, "%f", &ratio)
		return trace.TraceIDRatioBased(ratio)
	case "parentbased_always_on":
		return trace.ParentBased(trace.AlwaysSample())
	case "parentbased_always_off":
		return trace.ParentBased(trace.NeverSample())
	case "parentbased_traceidratio":
		var ratio float64 = 1.0
		fmt.Sscanf(arg, "%f", &ratio)
		return trace.ParentBased(trace.TraceIDRatioBased(ratio))
	default:
		return trace.ParentBased(trace.AlwaysSample())
	}
}

func logStartup(loc *time.Location, enabled bool, protocol, endpoint, sampler, samplerArg string) {
	entry := map[string]any{
		"ts":              time.Now().In(loc).Format(time.RFC3339Nano),
		"level":           "info",
		"msg":             "tracing_configured",
		"tracing_enabled": enabled,
	}
	if enabled {
		entry["otlp_protocol"] = protocol
		entry["otlp_endpoint"] = endpoint
		entry["sampler"] = sampler
		entry["sampler_arg"] = samplerArg
	}

	if b, err := json.Marshal(entry); err == nil {
		log.SetFlags(0)
		log.Println(string(b))
	}
}

func logError(loc *time.Location, err error) {
	entry := map[string]any{
		"ts":    time.Now().In(loc).Format(time.RFC3339Nano),
		"level": "error",
		"msg":   "tracing_init_failed",
		"error": err.Error(),
	}
	if b, err := json.Marshal(entry); err == nil {
		log.SetFlags(0)
		log.Println(string(b))
	}
}

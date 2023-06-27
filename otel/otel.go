package otel

import (
	"context"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// ShutdownFunc is a delegate that shuts down the OpenTelemetry components.
type ShutdownFunc func(ctx context.Context) error

// Run sets the global OpenTelemetry tracer provider and meter provider
// configured to use the OTLP HTTP exporter that will send telemetry
// to a local OpenTelemetry Collector.
func Run(ctx context.Context, serviceName string) (ShutdownFunc, error) {
	// Initialized the returned shutdownFunc to no-op.
	shutdownFunc := func(ctx context.Context) error { return nil }

	// Create Resource.
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return shutdownFunc, err
	}

	// Create the OTLP exporters.
	traceExp, err := otlptracehttp.New(ctx, otlptracehttp.WithInsecure())
	if err != nil {
		return shutdownFunc, err
	}
	metricExp, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithInsecure())
	if err != nil {
		return shutdownFunc, err
	}

	// Create the TracerProvider.
	tp := trace.NewTracerProvider(
		// Record information about this application in an Resource.
		trace.WithResource(res),
		// Set traces exporter.
		trace.WithBatcher(traceExp),
	)

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)

	// Register W3C Trace Context propagator as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Create the MeterProvider.
	mp := metric.NewMeterProvider(
		// Record information about this application in an Resource.
		metric.WithResource(res),
		// Set metrics exporter.
		metric.WithReader(metric.NewPeriodicReader(metricExp)),
	)

	// Update the returned shutdownFunc that calls both providers'
	// shutdown methods and make sure that a non-nil error is returned
	// if any returneed an error.
	shutdownFunc = func(ctx context.Context) error {
		var retErr error
		if err := tp.Shutdown(ctx); err != nil {
			retErr = err
		}
		if err := mp.Shutdown(ctx); err != nil {
			retErr = err
		}
		return retErr
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetMeterProvider(mp)

	// Add runtime metrics instrumentation.
	if err := runtime.Start(); err != nil {
		return nil, err
	}

	// Return the Shutdown function so that it can be used by the caller to
	// send all the telemetry before the application closes.
	return shutdownFunc, nil
}

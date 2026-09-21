// Package otel initializes the OpenTelemetry SDK for metrics, logs, and
// traces. It is gated by cfg.Enabled — when disabled (the default, no
// collector configured yet), InitSDK is a no-op so enabling it later
// requires no code changes elsewhere.
package otel

import (
	"context"
	"errors"
	"fmt"

	"github.com/itsLeonB/cardstack/backend/internal/core/config"
	"github.com/itsLeonB/cardstack/backend/internal/core/logger"
	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const scopeName = "github.com/itsLeonB/cardstack/backend"

var Tracer trace.Tracer = noop.NewTracerProvider().Tracer("")

// InitSDK initializes the OpenTelemetry SDK for metrics, logs, and traces.
func InitSDK(ctx context.Context, cfg config.OTel) (func(context.Context) error, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build otel resource: %w", err)
	}

	var shutdownFuncs []func(context.Context) error
	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		if err != nil {
			return fmt.Errorf("error shutting down otel resources: %w", err)
		}
		return nil
	}

	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	for _, initFn := range []func(context.Context, *resource.Resource) (func(context.Context) error, error){
		initMetrics, initLogs, initTraces,
	} {
		shutdownFunc, err := initFn(ctx, res)
		if err != nil {
			if e := shutdown(ctx); e != nil {
				logger.Error(e)
			}
			return nil, err
		}
		shutdownFuncs = append(shutdownFuncs, shutdownFunc)
	}

	return shutdown, nil
}

func initMetrics(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	reader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric reader: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader), sdkmetric.WithResource(res))
	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

func initLogs(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	exporter, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)
	global.SetLoggerProvider(lp)

	return lp.Shutdown, nil
}

func initTraces(ctx context.Context, res *resource.Resource) (func(context.Context) error, error) {
	exporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	Tracer = tp.Tracer(scopeName)

	return tp.Shutdown, nil
}

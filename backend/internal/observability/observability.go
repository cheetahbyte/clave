package observability

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

var meter metric.Meter

var meterProvider *sdkmetric.MeterProvider

func Meter() metric.Meter {
	if meter == nil {
		meter = otel.Meter("clave-api")
	}
	return meter
}

type Config struct {
	Enabled        bool
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Headers        map[string]string
	ExportInterval time.Duration
}

func DefaultConfig() Config {
	return Config{
		ServiceName:    "clave-api",
		ServiceVersion: "",
		Environment:    "development",
		ExportInterval: 30 * time.Second,
	}
}

func Init(ctx context.Context, cfg Config) (*sdkmetric.MeterProvider, error) {
	if !cfg.Enabled || cfg.Endpoint == "" {
		slog.Info("OpenTelemetry metrics disabled (set OTEL_ENABLED=true and OTEL_EXPORTER_OTLP_ENDPOINT)")
		return nil, nil
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironmentName(cfg.Environment),
		),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, err
	}

	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(cfg.Endpoint),
		otlpmetrichttp.WithInsecure(),
	}
	if len(cfg.Headers) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(cfg.Headers))
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(cfg.ExportInterval),
			),
		),
	)
	otel.SetMeterProvider(mp)
	meterProvider = mp
	meter = mp.Meter(cfg.ServiceName)

	slog.Info("OpenTelemetry metrics enabled",
		"service", cfg.ServiceName,
		"environment", cfg.Environment,
		"endpoint", cfg.Endpoint,
	)
	return mp, nil
}

func InitMetrics() {
	m := Meter()
	initHTTPMetrics(m)
	initDBMetrics(m)
	initBusinessMetrics(m)
}

func Shutdown(ctx context.Context) {
	if meterProvider != nil {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := meterProvider.Shutdown(ctx); err != nil {
			slog.Error("OpenTelemetry shutdown failed", "err", err)
		}
	}
}

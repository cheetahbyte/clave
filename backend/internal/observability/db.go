package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric"
)

var (
	dbConnAcquired   metric.Int64Counter
	dbConnIdle       metric.Int64Gauge
	dbConnTotal      metric.Int64Gauge
	dbConnAcquireDur metric.Float64Histogram
	dbConnCanceled   metric.Int64Counter
	dbEmptyAcquire   metric.Int64Counter
)

func initDBMetrics(m metric.Meter) {
	var err error
	dbConnAcquired, err = m.Int64Counter("db.pool.conn_acquired_total",
		metric.WithDescription("Total connections acquired from the pool"),
	)
	if err != nil {
		panic(err)
	}
	dbConnIdle, err = m.Int64Gauge("db.pool.conn_idle",
		metric.WithDescription("Idle connections in the pool"),
	)
	if err != nil {
		panic(err)
	}
	dbConnTotal, err = m.Int64Gauge("db.pool.conn_total",
		metric.WithDescription("Total connections in the pool"),
	)
	if err != nil {
		panic(err)
	}
	dbConnAcquireDur, err = m.Float64Histogram("db.pool.conn_acquire_duration_seconds",
		metric.WithDescription("Connection acquire duration in seconds"),
	)
	if err != nil {
		panic(err)
	}
	dbConnCanceled, err = m.Int64Counter("db.pool.conn_canceled_total",
		metric.WithDescription("Total canceled connection acquires"),
	)
	if err != nil {
		panic(err)
	}
	dbEmptyAcquire, err = m.Int64Counter("db.pool.empty_acquire_total",
		metric.WithDescription("Total empty connection acquires"),
	)
	if err != nil {
		panic(err)
	}
}

func StartDBPoolMetrics(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	if meterProvider == nil {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := pool.Stat()
				bg := context.Background()

				dbConnAcquired.Add(bg, stats.AcquireCount())
				dbConnCanceled.Add(bg, stats.CanceledAcquireCount())
				dbEmptyAcquire.Add(bg, int64(stats.EmptyAcquireCount()))

				dbConnIdle.Record(bg, int64(stats.IdleConns()))
				dbConnTotal.Record(bg, int64(stats.TotalConns()))

				dbConnAcquireDur.Record(bg, stats.AcquireDuration().Seconds())
			}
		}
	}()
	slog.Info("database pool metrics started", "interval", interval)
}

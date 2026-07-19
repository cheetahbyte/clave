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
		var previousAcquired int64
		var previousCanceled int64
		var previousEmpty int64
		var previousAcquireDuration time.Duration
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := pool.Stat()
				bg := context.Background()

				acquired := stats.AcquireCount()
				canceled := stats.CanceledAcquireCount()
				empty := int64(stats.EmptyAcquireCount())
				acquireDuration := stats.AcquireDuration()
				dbConnAcquired.Add(bg, acquired-previousAcquired)
				dbConnCanceled.Add(bg, canceled-previousCanceled)
				dbEmptyAcquire.Add(bg, empty-previousEmpty)

				dbConnIdle.Record(bg, int64(stats.IdleConns()))
				dbConnTotal.Record(bg, int64(stats.TotalConns()))

				dbConnAcquireDur.Record(bg, (acquireDuration - previousAcquireDuration).Seconds())
				previousAcquired = acquired
				previousCanceled = canceled
				previousEmpty = empty
				previousAcquireDuration = acquireDuration
			}
		}
	}()
	slog.Info("database pool metrics started", "interval", interval)
}

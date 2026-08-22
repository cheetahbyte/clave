package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	bizLicensesCreated        metric.Int64Counter
	bizLicenseValidations     metric.Int64Counter
	bizActivationAttempts     metric.Int64Counter
	bizTrialAttempts          metric.Int64Counter
	bizUpdateChecks           metric.Int64Counter
	bizArtifactDownloads      metric.Int64Counter
	bizAuditEvents            metric.Int64Counter
	bizUpdateCheckTelemetry   metric.Int64Counter
	bizClientCheckinTelemetry metric.Int64Counter
	bizRetentionCleanup       metric.Int64Counter
	bizDeltaJobs              metric.Int64Counter
	bizDeltaPatchRatio        metric.Float64Histogram
)

func initBusinessMetrics(m metric.Meter) {
	var err error
	bizLicensesCreated, err = m.Int64Counter("biz.licenses_created_total",
		metric.WithDescription("Total licenses created"),
	)
	if err != nil {
		panic(err)
	}
	bizLicenseValidations, err = m.Int64Counter("biz.license_validations_total",
		metric.WithDescription("Total license validations"),
	)
	if err != nil {
		panic(err)
	}
	bizActivationAttempts, err = m.Int64Counter("biz.activation_attempts_total",
		metric.WithDescription("Total activation attempts"),
	)
	if err != nil {
		panic(err)
	}
	bizTrialAttempts, err = m.Int64Counter("biz.trial_attempts_total",
		metric.WithDescription("Total trial attempts"),
	)
	if err != nil {
		panic(err)
	}
	bizUpdateChecks, err = m.Int64Counter("biz.update_checks_total",
		metric.WithDescription("Total update checks"),
	)
	if err != nil {
		panic(err)
	}
	bizArtifactDownloads, err = m.Int64Counter("biz.artifact_downloads_total",
		metric.WithDescription("Total artifact downloads"),
	)
	if err != nil {
		panic(err)
	}
	bizAuditEvents, err = m.Int64Counter("biz.audit_events_total",
		metric.WithDescription("Total audit events"),
	)
	if err != nil {
		panic(err)
	}
	bizUpdateCheckTelemetry, err = m.Int64Counter("biz.update_check_telemetry_total",
		metric.WithDescription("Update-check telemetry recorder outcomes"))
	if err != nil {
		panic(err)
	}
	bizClientCheckinTelemetry, err = m.Int64Counter("biz.client_checkin_telemetry_total",
		metric.WithDescription("Client check-in telemetry recorder outcomes"))
	if err != nil {
		panic(err)
	}
	bizRetentionCleanup, err = m.Int64Counter("biz.retention_cleanup_runs_total",
		metric.WithDescription("Retention cleanup operation outcomes"))
	if err != nil {
		panic(err)
	}
	bizDeltaJobs, err = m.Int64Counter("biz.delta_jobs_total", metric.WithDescription("Delta job state transitions"))
	if err != nil {
		panic(err)
	}
	bizDeltaPatchRatio, err = m.Float64Histogram("biz.delta_patch_ratio", metric.WithDescription("Delta patch bytes divided by target artifact bytes"))
	if err != nil {
		panic(err)
	}
}

func CountRetentionCleanup(ctx context.Context, dataset, outcome string) {
	if bizRetentionCleanup != nil {
		bizRetentionCleanup.Add(ctx, 1, metric.WithAttributes(
			attribute.String("dataset", dataset),
			attribute.String("outcome", outcome),
		))
	}
}

func CountDeltaJob(ctx context.Context, status string) {
	if bizDeltaJobs != nil {
		bizDeltaJobs.Add(ctx, 1, metric.WithAttributes(attribute.String("status", status)))
	}
}

func RecordDeltaPatchRatio(ctx context.Context, patchSize, targetSize int64) {
	if bizDeltaPatchRatio != nil && targetSize > 0 {
		bizDeltaPatchRatio.Record(ctx, float64(patchSize)/float64(targetSize))
	}
}

func CountClientCheckinTelemetry(ctx context.Context, outcome string) {
	if bizClientCheckinTelemetry == nil {
		return
	}
	bizClientCheckinTelemetry.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}

func CountUpdateCheckTelemetry(ctx context.Context, outcome string) {
	if bizUpdateCheckTelemetry == nil {
		return
	}
	bizUpdateCheckTelemetry.Add(ctx, 1, metric.WithAttributes(attribute.String("outcome", outcome)))
}

func CountLicenseCreated(ctx context.Context, outcome string) {
	if bizLicensesCreated == nil {
		return
	}
	bizLicensesCreated.Add(ctx, 1, metric.WithAttributes(
		attribute.String("outcome", outcome),
	))
}

func CountLicenseValidation(ctx context.Context, outcome string) {
	if bizLicenseValidations == nil {
		return
	}
	bizLicenseValidations.Add(ctx, 1, metric.WithAttributes(
		attribute.String("outcome", outcome),
	))
}

func CountActivationAttempt(ctx context.Context, outcome, reason string) {
	if bizActivationAttempts == nil {
		return
	}
	bizActivationAttempts.Add(ctx, 1, metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
	))
}

func CountTrialAttempt(ctx context.Context, outcome, reason string) {
	if bizTrialAttempts == nil {
		return
	}
	bizTrialAttempts.Add(ctx, 1, metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason", reason),
	))
}

func CountUpdateCheck(ctx context.Context, outcome string) {
	if bizUpdateChecks == nil {
		return
	}
	bizUpdateChecks.Add(ctx, 1, metric.WithAttributes(
		attribute.String("outcome", outcome),
	))
}

func CountArtifactDownload(ctx context.Context, outcome string) {
	if bizArtifactDownloads == nil {
		return
	}
	bizArtifactDownloads.Add(ctx, 1, metric.WithAttributes(
		attribute.String("outcome", outcome),
	))
}

func CountAuditEvent(ctx context.Context, action string) {
	if bizAuditEvents == nil {
		return
	}
	bizAuditEvents.Add(ctx, 1, metric.WithAttributes(
		attribute.String("action", action),
	))
}

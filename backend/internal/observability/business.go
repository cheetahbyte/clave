package observability

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	bizLicensesCreated    metric.Int64Counter
	bizLicenseValidations metric.Int64Counter
	bizActivationAttempts metric.Int64Counter
	bizTrialAttempts      metric.Int64Counter
	bizUpdateChecks       metric.Int64Counter
	bizArtifactDownloads  metric.Int64Counter
	bizAuditEvents        metric.Int64Counter
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

package retention

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/observability"
)

const (
	cleanupInterval  = 24 * time.Hour
	operationTimeout = 30 * time.Second
)

type Policies struct {
	AuditMetadataDays int
	AuditLogDays      int
	UpdateCheckDays   int
}

type store interface {
	DeleteStaleSelfServiceTokens(context.Context) (int64, error)
	DeleteStaleOrganizationInvites(context.Context) (int64, error)
	DeleteStaleAdminEmailCodes(context.Context) (int64, error)
	ScrubStaleAuditSecurityMetadata(context.Context, int32) (int64, error)
	DeleteStaleAuditLogs(context.Context, int32) (int64, error)
	DeleteStaleUpdateChecks(context.Context, int32) (int64, error)
}

type Worker struct {
	store    store
	policies Policies
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewWorker(q *db.Queries, policies Policies) *Worker {
	return newWorker(q, policies)
}

func newWorker(store store, policies Policies) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	worker := &Worker{store: store, policies: policies, cancel: cancel}
	worker.wg.Add(1)
	go worker.run(ctx)
	return worker
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	w.cleanup(ctx)
	for {
		select {
		case <-ticker.C:
			w.cleanup(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) cleanup(ctx context.Context) {
	operations := []struct {
		name string
		run  func(context.Context) (int64, error)
	}{
		{"self_service_tokens", w.store.DeleteStaleSelfServiceTokens},
		{"organization_invites", w.store.DeleteStaleOrganizationInvites},
		{"admin_email_codes", w.store.DeleteStaleAdminEmailCodes},
		{"audit_security_metadata", func(ctx context.Context) (int64, error) {
			return w.store.ScrubStaleAuditSecurityMetadata(ctx, int32(w.policies.AuditMetadataDays))
		}},
		{"audit_logs", func(ctx context.Context) (int64, error) {
			return w.store.DeleteStaleAuditLogs(ctx, int32(w.policies.AuditLogDays))
		}},
		{"update_checks", func(ctx context.Context) (int64, error) {
			return w.store.DeleteStaleUpdateChecks(ctx, int32(w.policies.UpdateCheckDays))
		}},
	}
	for _, operation := range operations {
		if ctx.Err() != nil {
			return
		}
		operationCtx, cancel := context.WithTimeout(ctx, operationTimeout)
		rows, err := operation.run(operationCtx)
		cancel()
		if err != nil {
			slog.Warn("retention cleanup failed", "dataset", operation.name, "err", err)
			observability.CountRetentionCleanup(ctx, operation.name, "error")
			continue
		}
		slog.Info("retention cleanup completed", "dataset", operation.name, "rows", rows)
		observability.CountRetentionCleanup(ctx, operation.name, "success")
	}
}

func (w *Worker) Close(ctx context.Context) {
	w.cancel()
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

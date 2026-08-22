package retention

import (
	"context"
	"errors"
	"testing"
)

type storeStub struct {
	calls        []string
	inviteErr    error
	metadataDays int32
	auditDays    int32
	updateDays   int32
}

func (s *storeStub) called(name string, err error) (int64, error) {
	s.calls = append(s.calls, name)
	return 2, err
}

func (s *storeStub) DeleteStaleSelfServiceTokens(context.Context) (int64, error) {
	return s.called("self_service_tokens", nil)
}
func (s *storeStub) DeleteStaleOrganizationInvites(context.Context) (int64, error) {
	return s.called("organization_invites", s.inviteErr)
}
func (s *storeStub) DeleteStaleAdminEmailCodes(context.Context) (int64, error) {
	return s.called("admin_email_codes", nil)
}
func (s *storeStub) ScrubStaleAuditSecurityMetadata(_ context.Context, days int32) (int64, error) {
	s.metadataDays = days
	return s.called("audit_security_metadata", nil)
}
func (s *storeStub) DeleteStaleAuditLogs(_ context.Context, days int32) (int64, error) {
	s.auditDays = days
	return s.called("audit_logs", nil)
}
func (s *storeStub) DeleteStaleUpdateChecks(_ context.Context, days int32) (int64, error) {
	s.updateDays = days
	return s.called("update_checks", nil)
}

func TestCleanupContinuesAfterFailureAndUsesPolicies(t *testing.T) {
	store := &storeStub{inviteErr: errors.New("database unavailable")}
	worker := &Worker{store: store, policies: Policies{
		AuditMetadataDays: 90,
		AuditLogDays:      180,
		UpdateCheckDays:   90,
	}}

	worker.cleanup(t.Context())

	if len(store.calls) != 6 {
		t.Fatalf("calls = %v", store.calls)
	}
	if store.metadataDays != 90 || store.auditDays != 180 || store.updateDays != 90 {
		t.Fatalf("policy days = metadata:%d audit:%d update:%d", store.metadataDays, store.auditDays, store.updateDays)
	}
}

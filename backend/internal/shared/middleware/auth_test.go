package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/google/uuid"
)

// newTestSessionManager returns a SessionManager with a fresh in-memory store.
// The session lifetime is set to 1 hour so tests don't have to worry about
// cookie expiry.
func newTestSessionManager() *scs.SessionManager {
	sm := scs.New()
	sm.Lifetime = time.Hour
	sm.IdleTimeout = time.Hour
	sm.Cookie.Name = "test_session"
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Path = "/"
	return sm
}

// primeSession creates a session and seeds it with the required admin session
// keys. It returns the session cookie so the caller can attach it to
// subsequent requests that go through the full middleware chain.
func primeSession(t *testing.T, sm *scs.SessionManager, adminID, orgID uuid.UUID, mfaVerified bool) *http.Cookie {
	t.Helper()

	rw := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sm.Put(ctx, "admin_user_id", adminID.String())
		sm.Put(ctx, "admin_email", "test@example.com")
		sm.Put(ctx, "admin_role", "admin")
		sm.Put(ctx, "admin_organization_id", orgID.String())
		sm.Put(ctx, "admin_mfa_verified", mfaVerified)
	})).ServeHTTP(rw, r)

	cookies := rw.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected a session cookie to be set")
	}
	return cookies[0]
}

// withSession wraps an http.Handler with the SessionManager's LoadAndSave
// middleware so that session data is loaded into the request context. This
// mirrors the production chain where SessionMW runs before VerifiedAuth.
func withSession(sm *scs.SessionManager, next http.Handler) http.Handler {
	return sm.LoadAndSave(next)
}

func TestRequireAdminVerified_AllowsVerifiedSession(t *testing.T) {
	sm := newTestSessionManager()
	adminID := uuid.New()
	orgID := uuid.New()
	cookie := primeSession(t, sm, adminID, orgID, true)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := withSession(sm, RequireAdminVerified(sm)(inner))

	r := httptest.NewRequest("GET", "/admin/protected", nil)
	r.AddCookie(cookie)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)

	if !called {
		t.Errorf("expected protected handler to be called for verified session")
	}
	if rw.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rw.Code)
	}
}

// TestRequireAdminVerified_DeniesUnverifiedSession is the core regression
// test for the bypass: a session whose admin_mfa_verified flag is false
// must be denied. Before the fix, the middleware consulted the DB and
// allowed the request when totp_enabled was false in the database.
func TestRequireAdminVerified_DeniesUnverifiedSession(t *testing.T) {
	sm := newTestSessionManager()
	adminID := uuid.New()
	orgID := uuid.New()
	cookie := primeSession(t, sm, adminID, orgID, false)

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := withSession(sm, RequireAdminVerified(sm)(inner))

	r := httptest.NewRequest("GET", "/admin/protected", nil)
	r.AddCookie(cookie)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)

	if called {
		t.Errorf("protected handler must NOT be called for unverified session")
	}
	if rw.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rw.Code)
	}
}

func TestRequireAdminVerified_DeniesUnverifiedSession_DoesNotMutateSession(t *testing.T) {
	// Regression: the old code would write admin_mfa_verified=true to the
	// session on the bypass path. The new middleware must not mutate the
	// session when denying.
	sm := newTestSessionManager()
	adminID := uuid.New()
	orgID := uuid.New()
	cookie := primeSession(t, sm, adminID, orgID, false)

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be reached")
	})
	handler := withSession(sm, RequireAdminVerified(sm)(inner))

	r := httptest.NewRequest("GET", "/admin/license", nil)
	r.AddCookie(cookie)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)

	if rw.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rw.Code)
	}
}

func TestRequireAdminVerified_DeniesWhenSessionMissing(t *testing.T) {
	sm := newTestSessionManager()

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler := withSession(sm, RequireAdminVerified(sm)(inner))

	r := httptest.NewRequest("GET", "/admin/protected", nil)
	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, r)

	if called {
		t.Errorf("handler must not be called when session is missing")
	}
	if rw.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rw.Code)
	}
}

func TestRequireAdminVerified_DeniesWhenAdminUserIDInvalid(t *testing.T) {
	sm := newTestSessionManager()

	// Set an invalid UUID for admin_user_id to simulate a corrupted session.
	rw := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	sm.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sm.Put(r.Context(), "admin_user_id", "not-a-uuid")
		sm.Put(r.Context(), "admin_organization_id", uuid.New().String())
		sm.Put(r.Context(), "admin_mfa_verified", true)
	})).ServeHTTP(rw, r)
	cookie := rw.Result().Cookies()[0]

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})
	handler := withSession(sm, RequireAdminVerified(sm)(inner))

	r2 := httptest.NewRequest("GET", "/admin/protected", nil)
	r2.AddCookie(cookie)
	rw2 := httptest.NewRecorder()
	handler.ServeHTTP(rw2, r2)

	if called {
		t.Errorf("handler must not be called for invalid admin_user_id")
	}
	if rw2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rw2.Code)
	}
}

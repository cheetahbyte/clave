package events

import (
	"testing"
	"time"
)

func TestEventExpirationMatchesCredentialLifecycle(t *testing.T) {
	tests := []struct {
		typeName string
		want     time.Duration
	}{
		{"admin.2fa_code", 10 * time.Minute},
		{"selfservice.magic_link", 15 * time.Minute},
		{"organization.invite", 7 * 24 * time.Hour},
		{"license.created", 7 * 24 * time.Hour},
		{"license.replaced", 7 * 24 * time.Hour},
		{"unknown", 0},
	}
	for _, test := range tests {
		t.Run(test.typeName, func(t *testing.T) {
			if got := eventExpiration(EmailEvent{Type: test.typeName}); got != test.want {
				t.Fatalf("expiration = %s, want %s", got, test.want)
			}
		})
	}
	if got := eventExpiration(DeltaGenerateEvent{}); got != 0 {
		t.Fatalf("delta expiration = %s, want none", got)
	}
}

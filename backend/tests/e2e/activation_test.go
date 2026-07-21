package e2e

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createClientLicense(t *testing.T, pool *pgxpool.Pool, orgID, productID uuid.UUID, licenseKey string) uuid.UUID {
	t.Helper()
	secret := os.Getenv("LICENSE_HMAC_SECRET")
	if secret == "" {
		t.Fatal("LICENSE_HMAC_SECRET must be set to seed a client license")
	}
	hash, err := argon2id.CreateHash(licenseKey, argon2id.DefaultParams)
	if err != nil {
		t.Fatalf("hash license key: %v", err)
	}
	normalizedKey := strings.NewReplacer("-", "", " ", "").Replace(strings.ToUpper(strings.TrimSpace(licenseKey)))
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(normalizedKey))
	var id uuid.UUID
	err = pool.QueryRow(context.Background(), `
		insert into licenses (organization_id, product_id, lookup_digest, key_phc, customer_email, max_activations, expires_at)
		values ($1, $2, $3, $4, $5, 1, now() + interval '365 days') returning id`,
		orgID, productID, mac.Sum(nil), hash, uniqueEmail(),
	).Scan(&id)
	if err != nil {
		t.Fatalf("create client license: %v", err)
	}
	return id
}

func TestClientDeviceDeactivate(t *testing.T) {
	requireE2E(t)
	pool := newPool(t)
	orgID := orgIDBySlug(t, pool, "default")
	productID := createProduct(t, pool, orgID, uniqueName("E2E Client Deactivate"))
	licenseKey := "LIC-TEST-CLIENT-DEACTIVATE"
	licenseID := createClientLicense(t, pool, orgID, productID, licenseKey)
	hwid := "e2e-client-device"
	client := httpClient(t)

	resp := postJSON(t, client, "/api/v1/client/licenses/activate", map[string]any{
		"licenseKey": licenseKey,
		"productId":  productID.String(),
		"deviceId":   map[string]string{"hwid": hwid},
	})
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("activate status = %d, want 200", resp.StatusCode)
	}
	activation := decode[struct {
		ActivationID uuid.UUID `json:"activationId"`
		Token        string    `json:"token"`
	}](t, resp)

	postDeactivate := func(body map[string]string) *http.Response {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal deactivation: %v", err)
		}
		req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/client/licenses/deactivate", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("create deactivation request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+activation.Token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("deactivate request: %v", err)
		}
		return resp
	}

	resp = postJSON(t, client, "/api/v1/client/licenses/deactivate", map[string]string{
		"token":    activation.Token,
		"deviceId": hwid,
	})
	if resp.StatusCode == http.StatusOK {
		drain(resp)
		t.Fatal("deactivate with JSON token status = 200, want failure")
	}
	drain(resp)

	resp = postDeactivate(map[string]string{"deviceId": hwid})
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("deactivate status = %d, want 200", resp.StatusCode)
	}
	result := decode[struct {
		OK bool `json:"ok"`
	}](t, resp)
	if !result.OK {
		t.Fatal("deactivate response should be ok")
	}

	var deactivated bool
	var reason string
	err := pool.QueryRow(context.Background(), `
		select deactivated_at is not null, deactivation_reason from activations where id = $1 and license_id = $2`,
		activation.ActivationID, licenseID,
	).Scan(&deactivated, &reason)
	if err != nil {
		t.Fatalf("read deactivation: %v", err)
	}
	if !deactivated || reason != "client_unregistration" {
		t.Fatalf("deactivation = (%t, %q), want (true, %q)", deactivated, reason, "client_unregistration")
	}

	resp = postDeactivate(map[string]string{"deviceId": hwid})
	if resp.StatusCode == http.StatusOK {
		drain(resp)
		t.Fatal("repeated deactivate status = 200, want failure")
	}
	drain(resp)

	resp = postJSON(t, client, "/api/v1/client/licenses/activate", map[string]any{
		"licenseKey": licenseKey,
		"productId":  productID.String(),
		"deviceId":   map[string]string{"hwid": hwid},
	})
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("reactivate status = %d, want 200", resp.StatusCode)
	}
	drain(resp)
}

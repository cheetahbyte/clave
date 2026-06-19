package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"

	"github.com/google/uuid"
)

func newCookieJar(t *testing.T) *cookiejar.Jar {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	return jar
}

func postJSON(t *testing.T, c *http.Client, path string, body any) *http.Response {
	t.Helper()
	buf, _ := json.Marshal(body)
	resp, err := c.Post(baseURL+path, "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func doMethod(t *testing.T, c *http.Client, method, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, baseURL+path, nil)
	if err != nil {
		t.Fatalf("new request %s %s: %v", method, path, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return v
}

func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// authenticate runs request-token -> validate and returns a client whose cookie
// jar holds the self-service session. Requires SELF_SERVICE_RETURN_TOKEN=true on
// the server so the raw magic-link token is returned in the response.
func authenticate(t *testing.T, email, orgSlug string) *http.Client {
	t.Helper()
	c := httpClient(t)

	resp := postJSON(t, c, "/api/v1/self-service/auth/request-token", map[string]string{
		"email":   email,
		"orgSlug": orgSlug,
	})
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("request-token status = %d, want 200", resp.StatusCode)
	}
	tokenResp := decode[struct {
		Ok    bool   `json:"ok"`
		Token string `json:"token"`
	}](t, resp)
	if tokenResp.Token == "" {
		t.Fatal("request-token returned empty token; is SELF_SERVICE_RETURN_TOKEN=true set?")
	}

	resp = postJSON(t, c, "/api/v1/self-service/auth/validate", map[string]string{
		"token":   tokenResp.Token,
		"orgSlug": orgSlug,
	})
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("validate status = %d, want 200", resp.StatusCode)
	}
	drain(resp)
	return c
}

type apiLicense struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

type apiDevice struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestSelfServiceFullFlow(t *testing.T) {
	requireE2E(t)
	pool := newPool(t)

	orgID := orgIDBySlug(t, pool, "default")
	productID := createProduct(t, pool, orgID, uniqueName("E2E Suite"))
	email := uniqueEmail()
	licenseID := createLicense(t, pool, orgID, productID, email, 3)
	dev1 := createDevice(t, pool, licenseID, "E2E MacBook")
	createDevice(t, pool, licenseID, "E2E Workstation")

	c := authenticate(t, email, "default")

	// list licenses — seeded one present and active
	resp := doMethod(t, c, http.MethodGet, "/api/v1/self-service/licenses")
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("list licenses status = %d, want 200", resp.StatusCode)
	}
	licenses := decode[[]apiLicense](t, resp)
	found := false
	for _, l := range licenses {
		if l.ID == licenseID.String() {
			found = true
			if !l.IsActive {
				t.Error("seeded license should be active")
			}
		}
	}
	if !found {
		t.Fatalf("license %s not returned in list", licenseID)
	}

	// list devices — expect 2
	devicesPath := "/api/v1/self-service/licenses/" + licenseID.String() + "/devices"
	resp = doMethod(t, c, http.MethodGet, devicesPath)
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("list devices status = %d, want 200", resp.StatusCode)
	}
	devices := decode[[]apiDevice](t, resp)
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}

	// remove one device
	resp = doMethod(t, c, http.MethodDelete, devicesPath+"/"+dev1.String())
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("remove device status = %d, want 200", resp.StatusCode)
	}
	drain(resp)
	if !deviceExists(t, pool, dev1) {
		t.Error("device should remain in db after deactivation")
	}
	if deviceHasActiveActivation(t, pool, dev1) {
		t.Error("device activation should be deactivated")
	}

	// list devices again — expect 1
	resp = doMethod(t, c, http.MethodGet, devicesPath)
	devices = decode[[]apiDevice](t, resp)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device after removal, got %d", len(devices))
	}

	// revoke license
	resp = doMethod(t, c, http.MethodPost, "/api/v1/self-service/licenses/"+licenseID.String()+"/revoke")
	if resp.StatusCode != http.StatusOK {
		drain(resp)
		t.Fatalf("revoke status = %d, want 200", resp.StatusCode)
	}
	revokeResp := decode[struct {
		Ok           bool   `json:"ok"`
		NewLicenseId string `json:"newLicenseId"`
	}](t, resp)
	if !revokeResp.Ok {
		t.Error("revoke response should be ok")
	}
	if revokeResp.NewLicenseId == "" {
		t.Fatal("revoke response should include newLicenseId")
	}

	// old license should be inactive
	if licenseActive(t, pool, licenseID) {
		t.Error("old license should be inactive after revoke")
	}

	// new license should exist and be active
	newID, err := uuid.Parse(revokeResp.NewLicenseId)
	if err != nil {
		t.Fatalf("invalid new license id: %v", err)
	}
	if !licenseActive(t, pool, newID) {
		t.Error("new license should be active")
	}

	// verify new license has same customer_email and product
	var newEmail, oldEmail string
	var newProductID, oldProductID string
	pool.QueryRow(context.Background(),
		`select customer_email from licenses where id = $1`, newID).Scan(&newEmail)
	pool.QueryRow(context.Background(),
		`select customer_email from licenses where id = $1`, licenseID).Scan(&oldEmail)
	if newEmail != oldEmail {
		t.Error("new license should have same customer email")
	}
	pool.QueryRow(context.Background(),
		`select product_id from licenses where id = $1`, newID).Scan(&newProductID)
	pool.QueryRow(context.Background(),
		`select product_id from licenses where id = $1`, licenseID).Scan(&oldProductID)
	if newProductID != oldProductID {
		t.Error("new license should have same product")
	}
}

func TestSelfServiceRequiresAuth(t *testing.T) {
	requireE2E(t)
	c := httpClient(t) // no session cookie

	resp := doMethod(t, c, http.MethodGet, "/api/v1/self-service/licenses")
	defer drain(resp)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated list status = %d, want 401", resp.StatusCode)
	}
}

func TestSelfServiceCrossOrgIsolation(t *testing.T) {
	requireE2E(t)
	pool := newPool(t)

	// org A: the attacker's session
	orgA := orgIDBySlug(t, pool, "default")
	prodA := createProduct(t, pool, orgA, uniqueName("E2E Org A"))
	emailA := uniqueEmail()
	createLicense(t, pool, orgA, prodA, emailA, 3)

	// org B: victim resources, different org + email
	slugB := uniqueName("e2e-org")
	orgB := createOrg(t, pool, slugB)
	prodB := createProduct(t, pool, orgB, uniqueName("E2E Org B"))
	emailB := uniqueEmail()
	licB := createLicense(t, pool, orgB, prodB, emailB, 3)
	devB := createDevice(t, pool, licB, "Victim Device")

	// attacker authenticates against org A
	c := authenticate(t, emailA, "default")

	// attacker must not see org B's license
	resp := doMethod(t, c, http.MethodGet, "/api/v1/self-service/licenses")
	licenses := decode[[]apiLicense](t, resp)
	for _, l := range licenses {
		if l.ID == licB.String() {
			t.Fatal("attacker should not see another org's license")
		}
	}

	// attacker must not delete org B's device
	resp = doMethod(t, c, http.MethodDelete,
		"/api/v1/self-service/licenses/"+licB.String()+"/devices/"+devB.String())
	drain(resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-org device delete status = %d, want 404", resp.StatusCode)
	}
	if !deviceExists(t, pool, devB) {
		t.Error("victim device must still exist after cross-org delete attempt")
	}
	if !deviceHasActiveActivation(t, pool, devB) {
		t.Error("victim activation must remain active after cross-org delete attempt")
	}

	// attacker must not revoke org B's license
	resp = doMethod(t, c, http.MethodPost,
		"/api/v1/self-service/licenses/"+licB.String()+"/revoke")
	drain(resp)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("cross-org revoke status = %d, want 404", resp.StatusCode)
	}
	if !licenseActive(t, pool, licB) {
		t.Error("victim license must remain active after cross-org revoke attempt")
	}
}

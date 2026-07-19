package main

import "testing"

func TestLoadConfigDerivesMemorySafeLimit(t *testing.T) {
	t.Setenv("WORKER_TOKEN", "token")
	t.Setenv("DELTA_MAX_ARTIFACT_BYTES", "67108864")
	t.Setenv("DELTA_MEMORY_BUDGET_BYTES", "1073741824")
	config, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	want := int64(1073741824 / 17)
	if config.EffectiveMaxBytes != want {
		t.Fatalf("effective limit = %d, want %d", config.EffectiveMaxBytes, want)
	}
}

func TestLoadConfigRejectsMissingToken(t *testing.T) {
	t.Setenv("WORKER_TOKEN", "")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected missing token error")
	}
}

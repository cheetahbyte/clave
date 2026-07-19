package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProcessorCompletesVerifiedPatch(t *testing.T) {
	source := make([]byte, 128<<10)
	target := make([]byte, len(source))
	for i := range source {
		source[i], target[i] = 'A', 'A'
	}
	copy(target[64<<10:64<<10+32], []byte("a small deterministic target change"))
	sourceSHA := fmt.Sprintf("%x", sha256.Sum256(source))
	targetSHA := fmt.Sprintf("%x", sha256.Sum256(target))
	completed := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/claim", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(JobDetails{ID: "job", Status: "running", Schema: "clave.delta/v1", Algorithm: "bsdiff", SourceVersion: "1.0.0", TargetVersion: "1.1.0", SourceSHA256: sourceSHA, TargetSHA256: targetSHA, SourceSize: int64(len(source)), TargetSize: int64(len(target))})
	})
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/artifacts/source", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(source) })
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/artifacts/target", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(target) })
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/complete", func(w http.ResponseWriter, r *http.Request) {
		patch, _ := io.ReadAll(r.Body)
		if int64(len(patch)) >= int64(len(target))*70/100 {
			t.Errorf("patch was not worthwhile: %d", len(patch))
		}
		completed = true
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &APIClient{baseURL: server.URL, token: "token", http: &http.Client{Timeout: time.Minute}}
	outcome := NewProcessor(client, 1<<20).Process(t.Context(), DeltaGenerateEvent{Type: "delta.generate", JobID: "job"})
	if outcome.Requeue {
		t.Fatal("unexpected requeue")
	}
	if !completed {
		t.Fatal("worker did not complete the patch")
	}
}

func TestProcessorSkipsArtifactOverEffectiveLimit(t *testing.T) {
	skipped := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/claim", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(JobDetails{ID: "job", Status: "running", SourceSize: 2 << 20, TargetSize: 2 << 20})
	})
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/skip", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["reason"] != "artifact_too_large" {
			t.Errorf("unexpected reason: %v", body["reason"])
		}
		skipped = true
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &APIClient{baseURL: server.URL, token: "token", http: &http.Client{Timeout: time.Minute}}
	outcome := NewProcessor(client, 1<<20).Process(t.Context(), DeltaGenerateEvent{Type: "delta.generate", JobID: "job"})
	if outcome.Requeue || !skipped {
		t.Fatalf("outcome=%+v skipped=%v", outcome, skipped)
	}
}

func TestProcessorRequeuesTransientClaimFailure(t *testing.T) {
	requeued := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/claim", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/requeue", func(w http.ResponseWriter, r *http.Request) {
		requeued = true
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &APIClient{baseURL: server.URL, token: "token", http: &http.Client{Timeout: time.Minute}}

	outcome := NewProcessor(client, 1<<20).Process(t.Context(), DeltaGenerateEvent{Type: "delta.generate", JobID: "job"})
	if !outcome.Requeue || !requeued {
		t.Fatalf("outcome=%+v requeued=%v", outcome, requeued)
	}
}

func TestProcessorFailsJobWhenTempDirectoryCannotBeCreated(t *testing.T) {
	failed := false
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/claim", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(JobDetails{ID: "job", Status: "running", SourceSize: 1, TargetSize: 1})
	})
	mux.HandleFunc("/api/v1/worker/delta-jobs/job/fail", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["reason"] != "temp storage unavailable" {
			t.Errorf("unexpected reason: %v", body["reason"])
		}
		failed = true
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	client := &APIClient{baseURL: server.URL, token: "token", http: &http.Client{Timeout: time.Minute}}
	processor := NewProcessor(client, 1<<20)
	processor.makeTempDir = func() (string, error) { return "", errors.New("temp storage unavailable") }

	outcome := processor.Process(t.Context(), DeltaGenerateEvent{Type: "delta.generate", JobID: "job"})
	if outcome.Requeue || !failed {
		t.Fatalf("outcome=%+v failed=%v", outcome, failed)
	}
}

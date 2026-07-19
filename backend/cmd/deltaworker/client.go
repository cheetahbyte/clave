package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type APIError struct {
	Status int
	Op     string
}

func (e *APIError) Error() string { return fmt.Sprintf("%s returned HTTP %d", e.Op, e.Status) }
func (e *APIError) Transient() bool {
	return e.Status == 0 || e.Status == http.StatusTooManyRequests || e.Status >= 500
}

type JobDetails struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	Schema        string `json:"schema"`
	Algorithm     string `json:"algorithm"`
	SourceVersion string `json:"sourceVersion"`
	TargetVersion string `json:"targetVersion"`
	SourceSHA256  string `json:"sourceSha256"`
	TargetSHA256  string `json:"targetSha256"`
	SourceSize    int64  `json:"sourceSize"`
	TargetSize    int64  `json:"targetSize"`
}

type APIClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewAPIClient(config Config) *APIClient {
	return &APIClient{baseURL: config.APIURL, token: config.WorkerToken, http: &http.Client{Timeout: config.HTTPTimeout}}
}

func (c *APIClient) request(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return c.http.Do(req)
}

func (c *APIClient) Claim(ctx context.Context, jobID string) (JobDetails, error) {
	resp, err := c.request(ctx, http.MethodPost, "/api/v1/worker/delta-jobs/"+jobID+"/claim", nil)
	if err != nil {
		return JobDetails{}, &APIError{Op: "claim"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return JobDetails{}, &APIError{Status: resp.StatusCode, Op: "claim"}
	}
	var details JobDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return JobDetails{}, &APIError{Op: "decode claim response"}
	}
	return details, nil
}

func (c *APIClient) Download(ctx context.Context, jobID, role, path string, maxBytes int64) (int64, string, error) {
	resp, err := c.request(ctx, http.MethodGet, "/api/v1/worker/delta-jobs/"+jobID+"/artifacts/"+role, nil)
	if err != nil {
		return 0, "", &APIError{Op: "download " + role}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, "", &APIError{Status: resp.StatusCode, Op: "download " + role}
	}
	if resp.ContentLength > maxBytes {
		return 0, "", fmt.Errorf("%s artifact exceeds memory-safe limit", role)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	hasher := sha256.New()
	limited := io.LimitReader(resp.Body, maxBytes+1)
	size, err := io.Copy(io.MultiWriter(file, hasher), limited)
	if err != nil {
		return 0, "", err
	}
	if size > maxBytes {
		return 0, "", fmt.Errorf("%s artifact exceeds memory-safe limit", role)
	}
	return size, fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (c *APIClient) Complete(ctx context.Context, jobID, path, patchSHA string, patchSize int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/worker/delta-jobs/"+jobID+"/complete", file)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Patch-SHA256", patchSHA)
	req.Header.Set("X-Patch-Size", strconv.FormatInt(patchSize, 10))
	resp, err := c.http.Do(req)
	if err != nil {
		return &APIError{Op: "complete"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &APIError{Status: resp.StatusCode, Op: "complete"}
	}
	return nil
}

func (c *APIClient) Terminal(ctx context.Context, jobID, action, reason, patchSHA string, patchSize int64) error {
	body, _ := json.Marshal(map[string]any{"reason": reason, "patchSha256": patchSHA, "patchSize": patchSize})
	resp, err := c.request(ctx, http.MethodPost, "/api/v1/worker/delta-jobs/"+jobID+"/"+action, bytes.NewReader(body))
	if err != nil {
		return &APIError{Op: action}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &APIError{Status: resp.StatusCode, Op: action}
	}
	return nil
}

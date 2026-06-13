package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/cheetahbyte/clave/pkg/delta"
)

type DeltaJob struct {
	Type             string `json:"type"`
	JobID            string `json:"jobId"`
	OrganizationID   string `json:"organizationId"`
	ReleaseID        string `json:"releaseId"`
	SourceReleaseID  string `json:"sourceReleaseId"`
	SourceArtifactID string `json:"sourceArtifactId"`
	TargetArtifactID string `json:"targetArtifactId"`
}

type JobDetails struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	SourceArtifactURL string `json:"sourceArtifactUrl"`
	TargetArtifactURL string `json:"targetArtifactUrl"`
	SourceVersion     string `json:"sourceVersion"`
	SourceBuild       string `json:"sourceBuild,omitempty"`
	TargetVersion     string `json:"targetVersion"`
	TargetBuild       string `json:"targetBuild,omitempty"`
	Os                string `json:"os"`
	Arch              string `json:"arch"`
}

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8000"
	}
	workerToken := os.Getenv("WORKER_TOKEN")
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	log.Printf("delta-worker starting, api=%s", apiURL)

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("rabbitmq dial: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("rabbitmq channel: %v", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare("clave.events", "topic", true, false, false, false, nil); err != nil {
		log.Fatalf("declare exchange: %v", err)
	}

	q, err := ch.QueueDeclare("clave.delta.generate", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("declare queue: %v", err)
	}
	if err := ch.QueueBind(q.Name, "delta.generate", "clave.events", false, nil); err != nil {
		log.Fatalf("bind queue: %v", err)
	}

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("consume: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	log.Println("delta-worker ready, waiting for messages")

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			processMessage(apiURL, workerToken, ch, msg)
		case <-sig:
			log.Println("shutting down")
			return
		}
	}
}

func processMessage(apiURL, workerToken string, ch *amqp.Channel, msg amqp.Delivery) {
	var job DeltaJob
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		log.Printf("invalid message: %v", err)
		msg.Reject(false)
		return
	}

	log.Printf("processing job %s", job.JobID)

	client := &http.Client{Timeout: 5 * time.Minute}

	if err := workerPost(client, apiURL, workerToken, job.JobID, "started", nil); err != nil {
		log.Printf("mark started: %v", err)
		msg.Nack(false, true)
		return
	}

	details, err := getJobDetails(client, apiURL, workerToken, job.JobID)
	if err != nil {
		log.Printf("get job details: %v", err)
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}

	fromDir, err := os.MkdirTemp("", "clave-delta-from-")
	if err != nil {
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}
	defer os.RemoveAll(fromDir)

	toDir, err := os.MkdirTemp("", "clave-delta-to-")
	if err != nil {
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}
	defer os.RemoveAll(toDir)

	if err := downloadAndUnzip(client, workerToken, details.SourceArtifactURL, fromDir); err != nil {
		log.Printf("extract source: %v", err)
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}
	if err := downloadAndUnzip(client, workerToken, details.TargetArtifactURL, toDir); err != nil {
		log.Printf("extract target: %v", err)
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}

	fromManifest, fromManifestSHA, err := delta.BuildManifest(fromDir)
	if err != nil {
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}
	toManifest, toManifestSHA, err := delta.BuildManifest(toDir)
	if err != nil {
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}

	fromMeta := delta.ReleaseMeta{
		Version:        details.SourceVersion,
		Build:          details.SourceBuild,
		ManifestSHA256: fromManifestSHA,
	}
	toMeta := delta.ReleaseMeta{
		Version:        details.TargetVersion,
		Build:          details.TargetBuild,
		ManifestSHA256: toManifestSHA,
	}

	dm := delta.Diff(fromManifest, toManifest, fromMeta, toMeta)
	deltaBytes, err := delta.BuildDelta(dm, toManifest, toDir)
	if err != nil {
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}

	if err := completeJob(client, apiURL, workerToken, job.JobID, deltaBytes); err != nil {
		log.Printf("complete job: %v", err)
		failJob(client, apiURL, workerToken, job.JobID, err.Error())
		msg.Nack(false, false)
		return
	}

	log.Printf("job %s completed successfully", job.JobID)
	msg.Ack(false)
}

func getJobDetails(client *http.Client, apiURL, token, jobID string) (*JobDetails, error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/worker/delta-jobs/%s", apiURL, jobID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get job returned %d: %s", resp.StatusCode, string(body))
	}
	var d JobDetails
	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return nil, fmt.Errorf("decode job: %w", err)
	}
	return &d, nil
}

func workerPost(client *http.Client, apiURL, token, jobID, action string, body []byte) error {
	url := fmt.Sprintf("%s/api/v1/worker/delta-jobs/%s/%s", apiURL, jobID, action)
	var r io.Reader
	if len(body) > 0 {
		r = bytes.NewReader(body)
	}
	req, _ := http.NewRequest("POST", url, r)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/octet-stream")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("worker api returned %d for %s", resp.StatusCode, action)
	}
	return nil
}

func completeJob(client *http.Client, apiURL, token, jobID string, deltaData []byte) error {
	return workerPost(client, apiURL, token, jobID, "complete", deltaData)
}

func failJob(client *http.Client, apiURL, token, jobID, errMsg string) {
	body, _ := json.Marshal(map[string]string{"error": errMsg})
	url := fmt.Sprintf("%s/api/v1/worker/delta-jobs/%s/failed", apiURL, jobID)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to report job failure: %v", err)
		return
	}
	resp.Body.Close()
}

func downloadAndUnzip(client *http.Client, token, url, dest string) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("download %s returned %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("read zip: %w", err)
	}

	for _, f := range r.File {
		clean := filepath.Clean(filepath.FromSlash(f.Name))
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		path := filepath.Join(dest, clean)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, 0755)
			continue
		}
		os.MkdirAll(filepath.Dir(path), 0755)
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(path)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

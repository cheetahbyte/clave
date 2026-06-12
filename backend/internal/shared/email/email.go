package email

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type Message struct {
	subject string
	html    string
}

const (
	emailQueueSize   = 64
	emailWorkers     = 2
	emailSendTimeout = 15 * time.Second
)

type emailJob struct {
	to  string
	msg Message
}

type Sender struct {
	host     string
	port     string
	username string
	password string
	from     string
	jobs     chan emailJob
}

func NewSender(host, port, username, password, from string) *Sender {
	s := &Sender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		jobs:     make(chan emailJob, emailQueueSize),
	}
	for i := 0; i < emailWorkers; i++ {
		go s.worker()
	}
	return s
}

func (s *Sender) Enqueue(to string, msg Message) {
	select {
	case s.jobs <- emailJob{to: to, msg: msg}:
	default:
		slog.Error("email queue full, dropping message", "to", to, "subject", msg.subject)
	}
}

func (s *Sender) worker() {
	for job := range s.jobs {
		done := make(chan struct{})
		go func() {
			if err := s.Send(job.to, job.msg); err != nil {
				slog.Error("async email send failed", "to", job.to, "subject", job.msg.subject, "err", err)
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(emailSendTimeout):
			slog.Error("async email send timed out", "to", job.to, "subject", job.msg.subject)
		}
	}
}

func (s *Sender) Send(to string, msg Message) error {
	fromAddr, err := mail.ParseAddress(s.from)
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	toAddr, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}

	raw, err := buildMessage(fromAddr.String(), toAddr.String(), msg)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	if s.username != "" && s.password != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)

		tlsConfig := &tls.Config{
			ServerName: s.host,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("tls dial: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, s.host)
		if err != nil {
			return fmt.Errorf("smtp client: %w", err)
		}
		defer client.Quit()

		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		if err := client.Mail(fromAddr.Address); err != nil {
			return fmt.Errorf("smtp mail: %w", err)
		}
		if err := client.Rcpt(toAddr.Address); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		_, err = w.Write([]byte(raw))
		if err != nil {
			return fmt.Errorf("smtp write: %w", err)
		}
		return w.Close()
	}

	return smtp.SendMail(addr, nil, fromAddr.Address, []string{toAddr.Address}, []byte(raw))
}

func buildMessage(from, to string, msg Message) (string, error) {
	if strings.ContainsAny(from, "\r\n") || strings.ContainsAny(to, "\r\n") || strings.ContainsAny(msg.subject, "\r\n") {
		return "", fmt.Errorf("email headers must not contain CRLF")
	}

	enc := mime.QEncoding
	encodedSubject := enc.Encode("utf-8", msg.subject)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", to))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.html)
	return b.String(), nil
}

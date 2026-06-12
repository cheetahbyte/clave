package email

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"
)

type Sender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSender(host, port, username, password, from string) *Sender {
	return &Sender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *Sender) Send(to, subject, htmlBody string) error {
	fromAddr, err := mail.ParseAddress(s.from)
	if err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	toAddr, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid to address: %w", err)
	}

	msg, err := buildMessage(fromAddr.String(), toAddr.String(), subject, htmlBody)
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
		_, err = w.Write([]byte(msg))
		if err != nil {
			return fmt.Errorf("smtp write: %w", err)
		}
		return w.Close()
	}

	return smtp.SendMail(addr, nil, fromAddr.Address, []string{toAddr.Address}, []byte(msg))
}

func buildMessage(from, to, subject, htmlBody string) (string, error) {
	if strings.ContainsAny(from, "\r\n") || strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return "", fmt.Errorf("email headers must not contain CRLF")
	}

	enc := mime.QEncoding
	encodedSubject := enc.Encode("utf-8", subject)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("From: %s\r\n", from))
	b.WriteString(fmt.Sprintf("To: %s\r\n", to))
	b.WriteString(fmt.Sprintf("Subject: %s\r\n", encodedSubject))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(htmlBody)
	return b.String(), nil
}

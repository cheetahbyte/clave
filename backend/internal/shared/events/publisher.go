// Package events publishes domain events to RabbitMQ for downstream workers
// (e.g. the emailer service) to consume.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// exchange mirrors the topic exchange the emailer worker binds to. Events are
// routed by their Type (e.g. "license.created").
const exchange = "clave.events"

// EmailEvent is the wire shape consumed by the emailer worker. It must stay in
// sync with EmailEvent in emailer/src/templates.ts.
type EmailEvent struct {
	Type  string         `json:"type"`
	Email string         `json:"email"`
	Data  map[string]any `json:"data"`
}

type DeltaGenerateEvent struct {
	Type  string `json:"type"`
	JobID string `json:"jobId"`
}

// Publisher holds a lazily-established RabbitMQ connection and republishes on
// failure. It is safe for concurrent use.
type Publisher struct {
	url string

	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

// NewPublisher returns a publisher for the given AMQP URL. The connection is
// established lazily on the first publish.
func NewPublisher(url string) *Publisher {
	return &Publisher{url: url}
}

// channel returns a live channel, (re)connecting if needed. Caller must not
// hold p.mu.
func (p *Publisher) channel() (*amqp.Channel, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn != nil && !p.conn.IsClosed() && p.ch != nil && !p.ch.IsClosed() {
		return p.ch, nil
	}

	conn, err := amqp.Dial(p.url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("amqp channel: %w", err)
	}
	// durable topic exchange — matches the worker's assertExchange.
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}

	p.conn = conn
	p.ch = ch
	return ch, nil
}

// PublishEmail publishes an email event using its Type as the routing key.
func (p *Publisher) PublishEmail(ctx context.Context, ev EmailEvent) error {
	return p.publish(ctx, ev.Type, ev)
}

func (p *Publisher) PublishDeltaGenerate(ctx context.Context, jobID string) error {
	return p.publish(ctx, "delta.generate", DeltaGenerateEvent{Type: "delta.generate", JobID: jobID})
}

func (p *Publisher) publish(ctx context.Context, routingKey string, event any) error {
	ch, err := p.channel()
	if err != nil {
		return err
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return ch.PublishWithContext(pubCtx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Body:         body,
	})
}

// PublishLicenseCreated emits a "license.created" event for the emailer worker.
// Optional fields are omitted when empty so the worker applies its defaults.
func (p *Publisher) PublishLicenseCreated(ctx context.Context, email, licenseKey, productName, portalLink string, isTrial bool) error {
	data := map[string]any{"licenseKey": licenseKey}
	if productName != "" {
		data["productName"] = productName
	}
	if portalLink != "" {
		data["portalLink"] = portalLink
	}
	if isTrial {
		data["isTrial"] = true
	}
	return p.PublishEmail(ctx, EmailEvent{
		Type:  "license.created",
		Email: email,
		Data:  data,
	})
}

// PublishAdminTwoFactorCode emits an "admin.2fa_code" event for the emailer
// worker.
func (p *Publisher) PublishAdminTwoFactorCode(ctx context.Context, email, code string, ttlMinutes int) error {
	return p.PublishEmail(ctx, EmailEvent{
		Type:  "admin.2fa_code",
		Email: email,
		Data: map[string]any{
			"code":       code,
			"ttlMinutes": ttlMinutes,
		},
	})
}

// PublishSelfServiceMagicLink emits a "selfservice.magic_link" event for the
// emailer worker.
func (p *Publisher) PublishSelfServiceMagicLink(ctx context.Context, email, link string) error {
	return p.PublishEmail(ctx, EmailEvent{
		Type:  "selfservice.magic_link",
		Email: email,
		Data:  map[string]any{"link": link},
	})
}

// PublishOrganizationInvite emits an "organization.invite" event for the
// emailer worker.
func (p *Publisher) PublishOrganizationInvite(ctx context.Context, email, orgName, link string) error {
	data := map[string]any{"link": link}
	if orgName != "" {
		data["orgName"] = orgName
	}
	return p.PublishEmail(ctx, EmailEvent{
		Type:  "organization.invite",
		Email: email,
		Data:  data,
	})
}

// PublishLicenseReplaced emits a "license.replaced" event for the emailer
// worker. Optional fields are omitted when empty so the worker applies its
// defaults.
func (p *Publisher) PublishLicenseReplaced(ctx context.Context, email, licenseKey, productName, portalLink string) error {
	data := map[string]any{"licenseKey": licenseKey}
	if productName != "" {
		data["productName"] = productName
	}
	if portalLink != "" {
		data["portalLink"] = portalLink
	}
	return p.PublishEmail(ctx, EmailEvent{
		Type:  "license.replaced",
		Email: email,
		Data:  data,
	})
}

// Close tears down the channel and connection.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch != nil {
		p.ch.Close()
		p.ch = nil
	}
	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil
		return err
	}
	return nil
}

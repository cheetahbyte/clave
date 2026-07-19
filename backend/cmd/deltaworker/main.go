package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		slog.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	debug.SetMemoryLimit(config.MemoryBudgetBytes)
	connection, err := amqp.Dial(config.RabbitMQURL)
	if err != nil {
		slog.Error("connect RabbitMQ", "err", err)
		os.Exit(1)
	}
	defer connection.Close()
	channel, err := connection.Channel()
	if err != nil {
		slog.Error("open RabbitMQ channel", "err", err)
		os.Exit(1)
	}
	defer channel.Close()
	if err := channel.ExchangeDeclare("clave.events", "topic", true, false, false, false, nil); err != nil {
		slog.Error("declare exchange", "err", err)
		os.Exit(1)
	}
	queue, err := channel.QueueDeclare("clave.delta.generate", true, false, false, false, nil)
	if err != nil {
		slog.Error("declare queue", "err", err)
		os.Exit(1)
	}
	if err := channel.QueueBind(queue.Name, "delta.generate", "clave.events", false, nil); err != nil {
		slog.Error("bind queue", "err", err)
		os.Exit(1)
	}
	if err := channel.Qos(1, 0, false); err != nil {
		slog.Error("set prefetch", "err", err)
		os.Exit(1)
	}
	deliveries, err := channel.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		slog.Error("consume queue", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	processor := NewProcessor(NewAPIClient(config), config.EffectiveMaxBytes)
	slog.Info("delta worker ready", "effectiveMaxArtifactBytes", config.EffectiveMaxBytes, "memoryBudgetBytes", config.MemoryBudgetBytes)
	for {
		select {
		case <-ctx.Done():
			return
		case delivery, ok := <-deliveries:
			if !ok {
				return
			}
			var event DeltaGenerateEvent
			if err := json.Unmarshal(delivery.Body, &event); err != nil || event.Type != "delta.generate" || event.JobID == "" {
				_ = delivery.Reject(false)
				continue
			}
			outcome := processor.Process(ctx, event)
			if outcome.Requeue {
				_ = delivery.Nack(false, true)
			} else {
				_ = delivery.Ack(false)
			}
		}
	}
}

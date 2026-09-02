package rabbitmq

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

// VfxJobPublisher publishes VFX Jobs to the queue named by the contract.
type VfxJobPublisher struct {
	ch *amqp.Channel
}

func NewVfxJobPublisher(conn *amqp.Connection) *VfxJobPublisher {
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	// The arguments must match the vfx service's declare of the same queue
	// byte for byte, or RabbitMQ rejects the second one (406) and the
	// handoff breaks. Both sides derive them from the contract.
	_, err = ch.QueueDeclare(vfxjob.QueueName, true, false, false, false, vfxjob.QueueArgs())
	if err != nil {
		panic(err)
	}

	return &VfxJobPublisher{ch: ch}
}

func (p *VfxJobPublisher) Publish(job vfxjob.Job) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal vfx job: %w", err)
	}

	err = p.ch.Publish(
		"", vfxjob.QueueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish vfx job: %w", err)
	}
	return nil
}

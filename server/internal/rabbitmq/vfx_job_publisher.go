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

	_, err = ch.QueueDeclare(vfxjob.QueueName, true, false, false, false, nil)
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

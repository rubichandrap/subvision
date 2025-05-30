package rabbitmq

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

type GenerateVfxJobPublisher struct {
	ch *amqp.Channel
}

type GenerateVfxJobPayload struct {
	ObjectKey string                `json:"objectKey"`
	Segments  []transcriber.Segment `json:"segments"`
}

func NewGenerateVfxJobPublisher(conn *amqp.Connection) *GenerateVfxJobPublisher {
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	_, err = ch.QueueDeclare("generating_vfx_jobs", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	return &GenerateVfxJobPublisher{ch: ch}
}

func (p *GenerateVfxJobPublisher) Publish(payload GenerateVfxJobPayload) error {
	fmt.Printf("Publishing generating_vfx_jobs to message queue")

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = p.ch.Publish(
		"", "generating_vfx_jobs", false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish upload job: %w", err)
	}
	return nil
}

package rabbitmq

import (
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type UploadJobPublisher struct {
	ch *amqp.Channel
}

type UploadJobPayload struct {
	UploadID string            `json:"uploadID"`
	Meta     map[string]string `json:"meta"`
}

func NewUploadJobPublisher(conn *amqp.Connection) *UploadJobPublisher {
	ch, err := conn.Channel()
	if err != nil {
		panic(err)
	}

	_, err = ch.QueueDeclare("upload_jobs", true, false, false, false, nil)
	if err != nil {
		panic(err)
	}

	return &UploadJobPublisher{ch: ch}
}

func (p *UploadJobPublisher) Publish(payload UploadJobPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	err = p.ch.Publish(
		"", "upload_jobs", false, false,
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

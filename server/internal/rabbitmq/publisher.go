package rabbitmq

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	channel *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) *Publisher {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("Failed to open RabbitMQ channel:", err)
	}

	_, err = ch.QueueDeclare(
		"upload_jobs", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatal("Failed to declare queue:", err)
	}

	return &Publisher{channel: ch}
}

func (p *Publisher) PublishJob(uploadID string, meta map[string]string) error {
	body, err := json.Marshal(map[string]interface{}{
		"uploadID": uploadID,
		"meta":     meta,
	})
	if err != nil {
		return err
	}

	return p.channel.Publish("", "upload_jobs", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

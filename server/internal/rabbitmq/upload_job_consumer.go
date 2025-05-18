package rabbitmq

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type UploadJobConsumer struct {
	conn *amqp.Connection
}

func NewUploadJobConsumer(conn *amqp.Connection) *UploadJobConsumer {
	return &UploadJobConsumer{conn: conn}
}

func (c *UploadJobConsumer) Start(handler func(UploadJobPayload)) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare("upload_jobs", true, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume("upload_jobs", "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var payload UploadJobPayload
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				log.Println("[UploadJobConsumer] Failed to parse message:", err)
				continue
			}
			log.Printf("[UploadJobConsumer] Received: %+v", payload)
			handler(payload)
		}
	}()

	return nil
}

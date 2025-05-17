package rabbitmq

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type JobPayload struct {
	UploadID string            `json:"uploadID"`
	Meta     map[string]string `json:"meta"`
}

func StartConsumer(conn *amqp.Connection, handler func(JobPayload)) {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("RabbitMQ Channel Error:", err)
	}

	msgs, err := ch.Consume(
		"upload_jobs", "", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatal("Failed to consume:", err)
	}

	go func() {
		for d := range msgs {
			var payload JobPayload
			if err := json.Unmarshal(d.Body, &payload); err != nil {
				log.Println("Failed to parse job:", err)
				continue
			}
			log.Println("[Worker] Received job:", payload.UploadID)
			handler(payload)
		}
	}()
}

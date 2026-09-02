package rabbitmq

import (
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rubichandrap/subvision/server/internal/vfxjob"
)

// JobCompletedConsumer consumes the render outcomes the vfx service publishes.
type JobCompletedConsumer struct {
	conn *amqp.Connection
}

func NewJobCompletedConsumer(conn *amqp.Connection) *JobCompletedConsumer {
	return &JobCompletedConsumer{conn: conn}
}

func (c *JobCompletedConsumer) Start(handler func(event vfxjob.JobCompleted) error) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(vfxjob.CompletedQueueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(vfxjob.CompletedQueueName, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var event vfxjob.JobCompleted
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("[JobCompletedConsumer] Malformed job_completed event, dropping it: %v\nbody: %s", err, d.Body)
				continue
			}
			if err := handler(event); err != nil {
				// A dropped completion event would leave the job in-flight
				// forever, so the handler's failure is loud, never silent.
				log.Printf("[JobCompletedConsumer] Failed to record completion for upload %s: %v", event.UploadID, err)
			}
		}
	}()

	return nil
}

// JobFailedConsumer consumes the dead-letter reports the vfx service publishes.
type JobFailedConsumer struct {
	conn *amqp.Connection
}

func NewJobFailedConsumer(conn *amqp.Connection) *JobFailedConsumer {
	return &JobFailedConsumer{conn: conn}
}

func (c *JobFailedConsumer) Start(handler func(event vfxjob.JobFailed) error) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(vfxjob.FailedQueueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(vfxjob.FailedQueueName, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var event vfxjob.JobFailed
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("[JobFailedConsumer] Malformed job_failed event, dropping it: %v\nbody: %s", err, d.Body)
				continue
			}
			if err := handler(event); err != nil {
				log.Printf("[JobFailedConsumer] Failed to record failure for upload %s: %v", event.UploadID, err)
			}
		}
	}()

	return nil
}

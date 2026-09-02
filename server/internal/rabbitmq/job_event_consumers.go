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

func (c *JobCompletedConsumer) Start(handle func(event vfxjob.JobCompleted) (bool, error)) error {
	return consumeEvents(c.conn, vfxjob.CompletedQueueName, "JobCompletedConsumer", handle)
}

// JobFailedConsumer consumes the dead-letter reports the vfx service publishes.
type JobFailedConsumer struct {
	conn *amqp.Connection
}

func NewJobFailedConsumer(conn *amqp.Connection) *JobFailedConsumer {
	return &JobFailedConsumer{conn: conn}
}

func (c *JobFailedConsumer) Start(handle func(event vfxjob.JobFailed) (bool, error)) error {
	return consumeEvents(c.conn, vfxjob.FailedQueueName, "JobFailedConsumer", handle)
}

// consumeEvents consumes JSON events from a durable queue with manual
// acknowledgement: a handler that reports failure requeues the event (the
// broker redelivers once the failure — usually the job store — recovers), a
// malformed event is logged loudly and dropped, and everything else is
// acknowledged. A recorded=false event landed on an unknown or terminal job;
// that is logged and dropped, never retried.
func consumeEvents[T any](conn *amqp.Connection, queue string, logPrefix string, handle func(event T) (bool, error)) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			var event T
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("[%s] Malformed event on %q, dropping it: %v\nbody: %s", logPrefix, queue, err, d.Body)
				ackEvent(d, ch, logPrefix)
				continue
			}

			recorded, err := handle(event)
			if err != nil {
				// A lost event would leave its job in-flight forever, so a
				// handler failure is loud and requeued rather than acked.
				log.Printf("[%s] Handling event on %q failed, requeueing: %v", logPrefix, queue, err)
				if nerr := d.Nack(false, true); nerr != nil {
					log.Printf("[%s] Failed to requeue event on %q: %v", logPrefix, queue, nerr)
				}
				continue
			}
			if !recorded {
				log.Printf("[%s] Event on %q landed on an unknown or terminal job, dropping it", logPrefix, queue)
			}
			ackEvent(d, ch, logPrefix)
		}
	}()

	return nil
}

func ackEvent(d amqp.Delivery, ch *amqp.Channel, logPrefix string) {
	if err := d.Ack(false); err != nil {
		log.Printf("[%s] Failed to acknowledge event: %v", logPrefix, err)
	}
}

// Package vfxjob defines the VFX Job contract between the server and the vfx
// service: the queue name and the payload shape each live here, once, and
// every publisher and consumer derives from this definition. The vfx service
// mirrors it in vfx/src/contract.ts — a change here must be made there too.
package vfxjob

import (
	"github.com/rubichandrap/subvision/server/internal/editspec"
	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

// QueueName is the queue the server publishes VFX Jobs to and the vfx service
// consumes them from.
const QueueName = "vfx_jobs"

// DeadQueueName is the queue a rejected or repeatedly failing VFX Job is
// dead-lettered into. The vfx_jobs queue carries dead-letter arguments
// routing to it, so every declare of vfx_jobs must pass the same arguments.
const DeadQueueName = "vfx_jobs_dead"

// QueueArgs are the arguments every declare of QueueName must pass: dead
// letters route to DeadQueueName via the default exchange. The vfx service's
// declare (vfx/src/services/rabbitmq/rabbitmq-subscriber-service.ts) must
// stay byte-for-byte equivalent, or RabbitMQ rejects the second declare with
// 406 and severs the handoff.
func QueueArgs() map[string]any {
	return map[string]any{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": DeadQueueName,
	}
}

// CompletedQueueName is the queue the vfx service publishes JobCompleted
// events to and the server consumes them from.
const CompletedQueueName = "job_completed"

// FailedQueueName is the queue the vfx service publishes JobFailed events to
// and the server consumes them from.
const FailedQueueName = "job_failed"

// Job is the message that tells the vfx service to render a video. EditSpec
// is nil when the upload carried no Edit Spec: the vfx service then renders
// with its own defaults.
type Job struct {
	UploadID  string                `json:"uploadId"`
	ObjectKey string                `json:"objectKey"`
	Segments  []transcriber.Segment `json:"segments"`
	EditSpec  *editspec.Spec        `json:"editSpec,omitempty"`
}

// JobCompleted reports a rendered Output.
type JobCompleted struct {
	UploadID  string `json:"uploadId"`
	OutputKey string `json:"outputKey"`
}

// JobFailed reports a job that exhausted its attempts without rendering.
type JobFailed struct {
	UploadID string `json:"uploadId"`
	Reason   string `json:"reason"`
}

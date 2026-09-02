// Package vfxjob defines the VFX Job contract between the server and the vfx
// service: the queue name and the payload shape each live here, once, and
// every publisher and consumer derives from this definition. The vfx service
// mirrors it in vfx/src/contract.ts — a change here must be made there too.
package vfxjob

import "github.com/rubichandrap/subvision/server/internal/transcriber"

// QueueName is the queue the server publishes VFX Jobs to and the vfx service
// consumes them from.
const QueueName = "vfx_jobs"

// Job is the message that tells the vfx service to render a video.
type Job struct {
	UploadID  string                `json:"uploadId"`
	ObjectKey string                `json:"objectKey"`
	Segments  []transcriber.Segment `json:"segments"`
	// AnimationType is reserved by the contract; consumers must not act on it.
	AnimationType string `json:"animationType,omitempty"`
}

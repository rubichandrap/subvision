package vfxjob

import (
	"encoding/json"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/transcriber"
)

// The wire shape is the contract: the vfx service parses exactly these JSON
// field names (see vfx/src/contract.ts). A rename here without the mirror
// change there severs the pipeline — this test makes that loud.
func TestJobWireShape(t *testing.T) {
	body, err := json.Marshal(Job{
		UploadID:  "u1",
		ObjectKey: "uploads/u1",
		Segments: []transcriber.Segment{
			{Start: 0, End: 1.5, Text: "hello"},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"uploadId":"u1","objectKey":"uploads/u1","segments":[{"start":0,"end":1.5,"text":"hello"}]}`
	if string(body) != want {
		t.Errorf("wire shape changed:\n got: %s\nwant: %s", body, want)
	}
}

func TestQueueNameMatchesContract(t *testing.T) {
	if QueueName != "vfx_jobs" {
		t.Errorf("QueueName = %q, want %q (keep in sync with vfx/src/contract.ts)", QueueName, "vfx_jobs")
	}
}

func TestJobCompletedWireShape(t *testing.T) {
	body, err := json.Marshal(JobCompleted{UploadID: "u1", OutputKey: "outputs/u1"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"uploadId":"u1","outputKey":"outputs/u1"}`
	if string(body) != want {
		t.Errorf("wire shape changed:\n got: %s\nwant: %s", body, want)
	}
}

func TestJobFailedWireShape(t *testing.T) {
	body, err := json.Marshal(JobFailed{UploadID: "u1", Reason: "render exploded"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"uploadId":"u1","reason":"render exploded"}`
	if string(body) != want {
		t.Errorf("wire shape changed:\n got: %s\nwant: %s", body, want)
	}
}

func TestEventQueueNamesMatchContract(t *testing.T) {
	if CompletedQueueName != "job_completed" {
		t.Errorf("CompletedQueueName = %q, want %q (keep in sync with vfx/src/contract.ts)", CompletedQueueName, "job_completed")
	}
	if FailedQueueName != "job_failed" {
		t.Errorf("FailedQueueName = %q, want %q (keep in sync with vfx/src/contract.ts)", FailedQueueName, "job_failed")
	}
}

// QueueArgs must stay equivalent to the vfx service's assertQueue options
// (deadLetterExchange/deadLetterRoutingKey in the subscriber), or RabbitMQ
// rejects the second declare with 406 PRECONDITION_FAILED and whichever
// service boots second crashes.
func TestQueueArgsMatchContract(t *testing.T) {
	args := QueueArgs()
	if args["x-dead-letter-exchange"] != "" {
		t.Errorf("x-dead-letter-exchange = %v, want \"\"", args["x-dead-letter-exchange"])
	}
	if args["x-dead-letter-routing-key"] != DeadQueueName {
		t.Errorf("x-dead-letter-routing-key = %v, want %q", args["x-dead-letter-routing-key"], DeadQueueName)
	}
}

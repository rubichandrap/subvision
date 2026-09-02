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

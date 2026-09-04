package vfxjob

import (
	"encoding/json"
	"testing"

	"github.com/rubichandrap/subvision/server/internal/editspec"
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
			{Start: 0, End: 1.5, Text: "hello", Words: []transcriber.Word{{Text: "hello", Start: 0, End: 1.5}}},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"uploadId":"u1","objectKey":"uploads/u1","segments":[{"start":0,"end":1.5,"text":"hello","words":[{"text":"hello","start":0,"end":1.5}]}]}`
	if string(body) != want {
		t.Errorf("wire shape changed:\n got: %s\nwant: %s", body, want)
	}
}

// A job carrying an Edit Spec publishes it under the editSpec key, exactly as
// vfx/src/contract.ts parses it. The old reserved animationType field is gone.
func TestJobEditSpecWireShape(t *testing.T) {
	body, err := json.Marshal(Job{
		UploadID:  "u1",
		ObjectKey: "uploads/u1",
		Segments: []transcriber.Segment{
			{Start: 0, End: 1.5, Text: "hello", Words: []transcriber.Word{{Text: "hello", Start: 0, End: 1.5}}},
		},
		EditSpec: &editspec.Spec{
			Trim:      editspec.Trim{Start: 2, End: 9},
			Frame:     editspec.Frame{Preset: "9:16", Ratio: 0.5625, Zoom: 1.5, PanX: -0.5, PanY: 0},
			Animation: "pop",
			Style: &editspec.Style{
				FontFamily:        "Montserrat",
				FontSizeScale:     1.2,
				Color:             "#FFFFFF",
				OutlineWidth:      8,
				OutlineColor:      "#000000",
				BottomMargin:      0.1,
				Background:        "box",
				BackgroundOpacity: 0.5,
				Uppercase:         true,
				HighlightColor:    "#FACC15",
			},
			Captions: &editspec.Captions{WordsPerPage: 6},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	want := `{"uploadId":"u1","objectKey":"uploads/u1","segments":[{"start":0,"end":1.5,"text":"hello","words":[{"text":"hello","start":0,"end":1.5}]}],` +
		`"editSpec":{"trim":{"start":2,"end":9},` +
		`"frame":{"preset":"9:16","ratio":0.5625,"zoom":1.5,"panX":-0.5,"panY":0},` +
		`"animation":"pop",` +
		`"style":{"fontFamily":"Montserrat","fontSizeScale":1.2,"color":"#FFFFFF","outlineWidth":8,"outlineColor":"#000000","bottomMargin":0.1,"background":"box","backgroundOpacity":0.5,"uppercase":true,"highlightColor":"#FACC15"},"captions":{"wordsPerPage":6}}}`
	if string(body) != want {
		t.Errorf("editSpec wire shape changed:\n got: %s\nwant: %s", body, want)
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

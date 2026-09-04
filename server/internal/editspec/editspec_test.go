package editspec

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustParse(t *testing.T, raw string) *Spec {
	t.Helper()
	spec, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse(%s): %v", raw, err)
	}
	return spec
}

func validSpecJSON() string {
	return `{
		"trim": {"start": 1.5, "end": 10},
		"frame": {"preset": "9:16", "ratio": 0.5625, "zoom": 1.5, "panX": -0.5, "panY": 0},
		"animation": "pop",
		"style": {
			"fontFamily": "Montserrat", "fontSizeScale": 1.2, "color": "#FFFFFF",
			"outlineWidth": 8, "outlineColor": "#000000", "bottomMargin": 0.1,
			"background": "box", "backgroundOpacity": 0.5, "uppercase": true,
			"highlightColor": "#FACC15"
		}
	}`
}

func TestParseEmptyMetadataMeansNoEdit(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		spec, err := Parse(raw)
		if err != nil {
			t.Fatalf("Parse(%q): %v", raw, err)
		}
		if spec != nil {
			t.Errorf("Parse(%q) = %+v, want nil", raw, spec)
		}
	}
}

func TestParseValidSpec(t *testing.T) {
	spec := mustParse(t, validSpecJSON())
	if spec.Trim.Start != 1.5 || spec.Trim.End != 10 {
		t.Errorf("trim did not survive the round trip: %+v", spec.Trim)
	}
	if spec.Frame.Preset != "9:16" || spec.Frame.Zoom != 1.5 {
		t.Errorf("frame did not survive the round trip: %+v", spec.Frame)
	}
	if spec.Animation != "pop" {
		t.Errorf("animation = %q, want %q", spec.Animation, "pop")
	}
	if spec.Style == nil || spec.Style.FontFamily != "Montserrat" || !spec.Style.Uppercase {
		t.Errorf("style did not survive the round trip: %+v", spec.Style)
	}
}

func TestParseRejectsViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutator func(map[string]any)
		want    string
	}{
		{"negative trim start", func(j map[string]any) { j["trim"] = map[string]any{"start": -1} }, "trim.start"},
		{"trim end before start", func(j map[string]any) { j["trim"] = map[string]any{"start": 5, "end": 4} }, "trim.end"},
		{"unknown preset", func(j map[string]any) { j["frame"] = map[string]any{"preset": "21:9", "ratio": 2.33, "zoom": 1, "panX": 0, "panY": 0} }, "frame.preset"},
		{"ratio contradicting preset", func(j map[string]any) { j["frame"] = map[string]any{"preset": "9:16", "ratio": 1.7777, "zoom": 1, "panX": 0, "panY": 0} }, "does not match preset"},
		{"ratio out of range", func(j map[string]any) { j["frame"] = map[string]any{"preset": "free", "ratio": 9, "zoom": 1, "panX": 0, "panY": 0} }, "frame.ratio"},
		{"zoom out of range", func(j map[string]any) { j["frame"] = map[string]any{"preset": "free", "ratio": 1, "zoom": 0, "panX": 0, "panY": 0} }, "frame.zoom"},
		{"pan out of range", func(j map[string]any) { j["frame"] = map[string]any{"preset": "free", "ratio": 1, "zoom": 1, "panX": 2, "panY": 0} }, "frame.panX"},
		{"unknown animation", func(j map[string]any) { j["animation"] = "wiggle" }, "animation"},
		{"unknown font", func(j map[string]any) { j["style"].(map[string]any)["fontFamily"] = "Comic Sans" }, "fontFamily"},
		{"fontSizeScale out of range", func(j map[string]any) { j["style"].(map[string]any)["fontSizeScale"] = 0 }, "fontSizeScale"},
		{"outlineWidth out of range", func(j map[string]any) { j["style"].(map[string]any)["outlineWidth"] = 64 }, "outlineWidth"},
		{"bottomMargin out of range", func(j map[string]any) { j["style"].(map[string]any)["bottomMargin"] = 0.9 }, "bottomMargin"},
		{"backgroundOpacity out of range", func(j map[string]any) { j["style"].(map[string]any)["backgroundOpacity"] = 1.5 }, "backgroundOpacity"},
		{"unknown background", func(j map[string]any) { j["style"].(map[string]any)["background"] = "shadow" }, "background"},
		{"bad color", func(j map[string]any) { j["style"].(map[string]any)["color"] = "white" }, "style.color"},
		{"bad highlight color", func(j map[string]any) { j["style"].(map[string]any)["highlightColor"] = "#FFF" }, "highlightColor"},
		{"wordsPerPage too small", func(j map[string]any) { j["captions"] = map[string]any{"wordsPerPage": 1} }, "wordsPerPage"},
		{"wordsPerPage too large", func(j map[string]any) { j["captions"] = map[string]any{"wordsPerPage": 9} }, "wordsPerPage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var job map[string]any
			if err := json.Unmarshal([]byte(validSpecJSON()), &job); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			tt.mutator(job)
			raw, err := json.Marshal(job)
			if err != nil {
				t.Fatalf("marshal mutated job: %v", err)
			}
			_, parseErr := Parse(string(raw))
			if parseErr == nil {
				t.Fatalf("expected Parse to reject the spec")
			}
			if !strings.Contains(parseErr.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", parseErr.Error(), tt.want)
			}
		})
	}
}

func TestParseCaptionsOptional(t *testing.T) {
	spec := mustParse(t, validSpecJSON())
	if spec.Captions != nil {
		t.Errorf("Captions = %+v, want nil (absent knob stays absent)", spec.Captions)
	}
}

func TestParseCaptionsRoundTrip(t *testing.T) {
	var job map[string]any
	if err := json.Unmarshal([]byte(validSpecJSON()), &job); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	job["captions"] = map[string]any{"wordsPerPage": 6}
	raw, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal job: %v", err)
	}
	spec := mustParse(t, string(raw))
	if spec.Captions == nil || spec.Captions.WordsPerPage != 6 {
		t.Errorf("Captions = %+v, want {WordsPerPage: 6}", spec.Captions)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse("not json at all"); err == nil {
		t.Fatal("expected invalid JSON to be rejected")
	}
}

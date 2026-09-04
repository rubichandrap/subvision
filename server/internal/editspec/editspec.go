// Package editspec defines the Edit Spec the client configures in the editor
// and attaches to an upload as the "editSpec" tus metadata key: the trim
// window, the target Frame, the Animation, and the Subtitle Style. The vfx
// service mirrors this shape in vfx/src/contract.ts — a change here must be
// made there too. Parsing is fail-loud: a malformed spec is an error naming
// the violation, never silently reinterpreted.
package editspec

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Frame presets and their exact ratios (width / height). "free" carries the
// ratio in the payload instead.
const (
	PresetFree  = "free"
	PresetReel  = "9:16"
	PresetPost  = "4:5"
	PresetBox   = "1:1"
	PresetWide  = "16:9"
	maxFramePresetRatio = 3.5
)

var framePresetRatios = map[string]float64{
	PresetReel: 9.0 / 16.0,
	PresetPost: 4.0 / 5.0,
	PresetBox:  1.0,
	PresetWide: 16.0 / 9.0,
}

// Animations the vfx service can render. "random" must be resolved to one of
// these by the client; the contract never carries it.
var Animations = map[string]bool{
	"fade":    true,
	"slide":   true,
	"karaoke": true,
	"pop":     true,
}

// Fonts the vfx service bundles; anything else would render in a fallback face.
var Fonts = map[string]bool{
	"Montserrat": true,
	"Inter":      true,
	"Poppins":    true,
	"Oswald":     true,
	"Bebas Neue": true,
	"Anton":      true,
}

var backgrounds = map[string]bool{"none": true, "box": true}

var hexColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// Trim is the rendered window of the source video, in seconds. End == 0 (or
// absent) means "until the video ends".
type Trim struct {
	Start float64 `json:"start"`
	End   float64 `json:"end,omitempty"`
}

// Frame is the target canvas: an aspect ratio (from a Frame Preset or free)
// with the source video scaled and panned to fill it, crop-to-fill.
type Frame struct {
	Preset string  `json:"preset"`
	Ratio  float64 `json:"ratio"`            // width / height
	Zoom   float64 `json:"zoom"`             // 1..5, 1 = cover
	PanX   float64 `json:"panX"`             // -1..1 across the overflow
	PanY   float64 `json:"panY"`
}

// Style is the Subtitle Style of the rendered captions. When a Style is
// present, every field is required — the editor always sends the full set.
type Style struct {
	FontFamily        string  `json:"fontFamily"`
	FontSizeScale     float64 `json:"fontSizeScale"`     // 0.5..2
	Color             string  `json:"color"`             // #RRGGBB
	OutlineWidth      float64 `json:"outlineWidth"`      // 0..32 px at 1080-height reference
	OutlineColor      string  `json:"outlineColor"`      // #RRGGBB
	BottomMargin      float64 `json:"bottomMargin"`      // 0..0.8 of frame height
	Background        string  `json:"background"`        // none | box
	BackgroundOpacity float64 `json:"backgroundOpacity"` // 0..1
	Uppercase         bool    `json:"uppercase"`
	HighlightColor    string  `json:"highlightColor"` // #RRGGBB
}

// Spec is the complete Edit Spec. A nil *Spec means "no edit": the vfx
// service renders the full video with its defaults.
type Spec struct {
	Trim      Trim      `json:"trim"`
	Frame     Frame     `json:"frame"`
	Animation string    `json:"animation"`
	Style     *Style    `json:"style,omitempty"`
	Captions  *Captions `json:"captions,omitempty"`
}

// Captions carries the Caption Page size for the karaoke and pop
// animations. Nil means the job predates the knob: the vfx service renders
// page size 4.
type Captions struct {
	WordsPerPage int `json:"wordsPerPage"`
}

// Parse decodes and validates the raw editSpec metadata value. An empty
// value parses as (nil, nil) — "no edit" — while a malformed one returns an
// error naming the violation.
func Parse(raw string) (*Spec, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var spec Spec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return nil, fmt.Errorf("editSpec is not valid JSON: %w", err)
	}
	if err := spec.validate(); err != nil {
		return nil, err
	}
	return &spec, nil
}

func (s Spec) validate() error {
	if err := s.validateTrim(); err != nil {
		return err
	}
	if err := s.validateFrame(); err != nil {
		return err
	}
	if !Animations[s.Animation] {
		return fmt.Errorf(`editSpec.animation must be one of fade, slide, karaoke, pop, got %q`, s.Animation)
	}
	if s.Captions != nil {
		if s.Captions.WordsPerPage < 2 || s.Captions.WordsPerPage > 8 {
			return fmt.Errorf("editSpec.captions.wordsPerPage must be an integer within [2, 8], got %v", s.Captions.WordsPerPage)
		}
	}

	if s.Style == nil {
		return nil
	}
	return s.Style.validate()
}

func (s Spec) validateTrim() error {
	if !finite(s.Trim.Start) || s.Trim.Start < 0 {
		return fmt.Errorf("editSpec.trim.start must be a non-negative number of seconds, got %v", s.Trim.Start)
	}
	if s.Trim.End != 0 && (!finite(s.Trim.End) || s.Trim.End <= s.Trim.Start) {
		return fmt.Errorf("editSpec.trim.end must be 0 (render to the end) or greater than trim.start (%v), got %v", s.Trim.Start, s.Trim.End)
	}
	return nil
}

func (s Spec) validateFrame() error {
	f := s.Frame
	if !finite(f.Ratio) || f.Ratio <= 0 || f.Ratio > maxFramePresetRatio {
		return fmt.Errorf("editSpec.frame.ratio must be within (0, %v], got %v", maxFramePresetRatio, f.Ratio)
	}
	if preset, ok := framePresetRatios[f.Preset]; ok {
		// A preset's ratio must agree with the preset within 1%; anything
		// else means the client sent contradictory values.
		if math.Abs(f.Ratio-preset) > preset*0.01 {
			return fmt.Errorf("editSpec.frame.ratio %v does not match preset %q (ratio %v)", f.Ratio, f.Preset, preset)
		}
	} else if f.Preset != PresetFree {
		return fmt.Errorf("editSpec.frame.preset must be one of 9:16, 4:5, 1:1, 16:9, free, got %q", f.Preset)
	}
	if !finite(f.Zoom) || f.Zoom < 1 || f.Zoom > 5 {
		return fmt.Errorf("editSpec.frame.zoom must be within [1, 5], got %v", f.Zoom)
	}
	for name, value := range map[string]float64{"panX": f.PanX, "panY": f.PanY} {
		if !finite(value) || value < -1 || value > 1 {
			return fmt.Errorf("editSpec.frame.%s must be within [-1, 1], got %v", name, value)
		}
	}
	return nil
}

func (st Style) validate() error {
	if !Fonts[st.FontFamily] {
		return fmt.Errorf(`editSpec.style.fontFamily must be one of Montserrat, Inter, Poppins, Oswald, Bebas Neue, Anton, got %q`, st.FontFamily)
	}
	if !finite(st.FontSizeScale) || st.FontSizeScale < 0.5 || st.FontSizeScale > 2 {
		return fmt.Errorf("editSpec.style.fontSizeScale must be within [0.5, 2], got %v", st.FontSizeScale)
	}
	if !finite(st.OutlineWidth) || st.OutlineWidth < 0 || st.OutlineWidth > 32 {
		return fmt.Errorf("editSpec.style.outlineWidth must be within [0, 32], got %v", st.OutlineWidth)
	}
	if !finite(st.BottomMargin) || st.BottomMargin < 0 || st.BottomMargin > 0.8 {
		return fmt.Errorf("editSpec.style.bottomMargin must be within [0, 0.8], got %v", st.BottomMargin)
	}
	if !finite(st.BackgroundOpacity) || st.BackgroundOpacity < 0 || st.BackgroundOpacity > 1 {
		return fmt.Errorf("editSpec.style.backgroundOpacity must be within [0, 1], got %v", st.BackgroundOpacity)
	}
	if !backgrounds[st.Background] {
		return fmt.Errorf(`editSpec.style.background must be "none" or "box", got %q`, st.Background)
	}
	for name, value := range map[string]string{
		"color":          st.Color,
		"outlineColor":   st.OutlineColor,
		"highlightColor": st.HighlightColor,
	} {
		if !hexColor.MatchString(value) {
			return fmt.Errorf("editSpec.style.%s must be a #RRGGBB color, got %q", name, value)
		}
	}
	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

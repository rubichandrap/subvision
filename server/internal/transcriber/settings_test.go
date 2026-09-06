package transcriber

import "testing"

func TestVADRequired(t *testing.T) {
	cases := []struct {
		settings Settings
		want     bool
	}{
		{Settings{}, false},
		{Settings{VADModelPath: "models/ggml-silero-v5.1.2.bin"}, true},
	}
	for _, tc := range cases {
		if got := tc.settings.VADRequired(); got != tc.want {
			t.Errorf("Settings{%q}.VADRequired() = %v, want %v", tc.settings.VADModelPath, got, tc.want)
		}
	}
}

func TestDTWPresetFor(t *testing.T) {
	cases := []struct {
		modelPath string
		want      DTWPreset
		wantOK    bool
	}{
		{"models/ggml-base.en.bin", "base_en", true},
		{"models/ggml-small.bin", "small", true},
		{"models/ggml-tiny.en.bin", "tiny_en", true},
		{"models/ggml-large-v3-turbo.bin", "large_v3_turbo", true},
		{"ggml-base.en.bin", "base_en", true},
		// No upstream alignment-heads preset to match: DTW stays off rather
		// than guessing.
		{"models/custom-finetuned.bin", "", false},
		{"models/ggml-large.bin", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := DTWPresetFor(tc.modelPath)
		if ok != tc.wantOK || got != tc.want {
			t.Errorf("DTWPresetFor(%q) = (%q, %v), want (%q, %v)", tc.modelPath, got, ok, tc.want, tc.wantOK)
		}
	}
}

package transcriber

import "testing"

func TestGatingEnabled(t *testing.T) {
	cases := []struct {
		settings Settings
		want     bool
	}{
		{Settings{}, false},
		{Settings{SpeechGating: true}, true},
	}
	for _, tc := range cases {
		if got := tc.settings.GatingEnabled(); got != tc.want {
			t.Errorf("Settings{SpeechGating: %v}.GatingEnabled() = %v, want %v", tc.settings.SpeechGating, got, tc.want)
		}
	}
}

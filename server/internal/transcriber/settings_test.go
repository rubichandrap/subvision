package transcriber

import "testing"

func TestGatingEnabled(t *testing.T) {
	cases := []struct {
		settings Settings
		want     bool
	}{
		{Settings{}, false},
		{Settings{VADGating: true}, true},
	}
	for _, tc := range cases {
		if got := tc.settings.GatingEnabled(); got != tc.want {
			t.Errorf("Settings{VADGating: %v}.GatingEnabled() = %v, want %v", tc.settings.VADGating, got, tc.want)
		}
	}
}

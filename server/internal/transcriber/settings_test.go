package transcriber

import "testing"

func TestSettingsCarriesModelPath(t *testing.T) {
	s := Settings{ModelPath: "model.bin"}
	if s.ModelPath != "model.bin" {
		t.Errorf("Settings.ModelPath = %q, want %q", s.ModelPath, "model.bin")
	}
}

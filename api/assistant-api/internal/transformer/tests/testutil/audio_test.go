package transformer_testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpeechPCMFixture(t *testing.T) {
	fixture, err := os.Stat(filepath.Join(testdataDir(), "hello_world.pcm"))
	if err != nil {
		t.Fatalf("locate shipped speech fixture: %v", err)
	}
	audio := LoadSpeechPCM(t, "hello_world.pcm")
	if len(audio) == 0 || int64(len(audio)) != fixture.Size() || len(audio)%BytesPerSample != 0 {
		t.Fatalf("expected complete LINEAR16 fixture, received %d bytes", len(audio))
	}
}

package internal_exotel

import (
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
)

func TestAudioProcessor_AmbientConfigureAndIdleOutputFrame(t *testing.T) {
	audioProcessor, err := NewAudioProcessor(nil)
	if err != nil {
		t.Fatalf("NewAudioProcessor error: %v", err)
	}
	if audioProcessor.ambientMixer == nil {
		t.Fatal("expected ambient mixer to be initialized")
	}
	if audioProcessor.resampler == nil {
		t.Fatal("expected resampler")
	}

	err = audioProcessor.ConfigureAmbient(internal_ambient.NewConfig(internal_ambient.ProfileCafe, 18))
	if err != nil {
		t.Fatalf("ConfigureAmbient error: %v", err)
	}

	frame, ok := audioProcessor.IdleOutputFrame()
	if !ok {
		t.Fatal("expected idle output frame")
	}
	if len(frame.ProviderAudio) != OutputChunkSize {
		t.Fatalf("unexpected idle frame length: got=%d want=%d", len(frame.ProviderAudio), OutputChunkSize)
	}
}

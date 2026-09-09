package internal_vonage

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

	err = audioProcessor.ConfigureAmbient(internal_ambient.NewConfig(internal_ambient.ProfileCafe, 18))
	if err != nil {
		t.Fatalf("ConfigureAmbient error: %v", err)
	}
	currentAmbientConfig := audioProcessor.ambientMixer.CurrentConfig()
	if currentAmbientConfig.Profile != internal_ambient.ProfileCafe {
		t.Fatalf("unexpected ambient profile: got=%s want=%s", currentAmbientConfig.Profile, internal_ambient.ProfileCafe)
	}

	frame, ok := audioProcessor.IdleOutputFrame()
	if !ok {
		t.Fatal("expected idle output frame")
	}
	if len(frame.ProviderAudio) != OutputChunkSize {
		t.Fatalf("unexpected idle frame length: got=%d want=%d", len(frame.ProviderAudio), OutputChunkSize)
	}
}

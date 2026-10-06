package internal_vobiz

import (
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
)

func TestClearOutputBufferDiscardsStreamingTail(t *testing.T) {
	processor, err := NewAudioProcessor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer processor.inputWriter.Close()
	defer processor.outputWriter.Close()
	if err := processor.ProcessAssistantAudio(make([]byte, 802), false); err != nil {
		t.Fatal(err)
	}
	processor.ClearOutputBuffer()
	if err := processor.ProcessAssistantAudio(nil, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := processor.NextOutputFrame(); ok {
		t.Fatal("discarded filter tail was queued again")
	}
}

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
	if audioProcessor.OutputFrameDuration() != ChunkDuration {
		t.Fatalf("unexpected output frame duration: got=%s want=%s", audioProcessor.OutputFrameDuration(), ChunkDuration)
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

package internal_twilio

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/protos"
)

type twilioFakeResampler struct {
	out []byte
	err error
}

func (resampler *twilioFakeResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if resampler.err != nil {
		return nil, resampler.err
	}
	return append([]byte(nil), resampler.out...), nil
}

func (resampler *twilioFakeResampler) Close() {}

type twilioFakeMixer struct {
	err error
}

func (mixer *twilioFakeMixer) Configure(_ internal_ambient.Config) error { return nil }

func (mixer *twilioFakeMixer) Mix(primary []byte) ([]byte, error) {
	return primary, mixer.err
}

func (mixer *twilioFakeMixer) Reset() {}

func (mixer *twilioFakeMixer) CurrentConfig() internal_ambient.Config {
	return internal_ambient.Config{}
}

func TestAudioProcessor_ConvertOutputAudio_UsesResampler(t *testing.T) {
	audioProcessor := &AudioProcessor{
		resampler:        &twilioFakeResampler{out: []byte{1, 2, 3}},
		downstreamConfig: &protos.AudioConfig{},
		twilioConfig:     &protos.AudioConfig{},
	}

	got, err := audioProcessor.convertOutputAudio([]byte{9})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("unexpected output: %v", got)
	}
}

func TestAudioProcessor_ConvertOutputAudio_PropagatesError(t *testing.T) {
	audioProcessor := &AudioProcessor{
		resampler:        &twilioFakeResampler{err: errors.New("boom")},
		downstreamConfig: &protos.AudioConfig{},
		twilioConfig:     &protos.AudioConfig{},
	}

	if _, err := audioProcessor.convertOutputAudio([]byte{1}); err == nil {
		t.Fatal("expected error")
	}
}

func TestAudioProcessor_ApplyAmbient_NilMixerPassthrough(t *testing.T) {
	audioProcessor := &AudioProcessor{}
	input := []byte{1, 2, 3}

	output := audioProcessor.applyAmbient(input)
	if len(output) != len(input) {
		t.Fatalf("expected passthrough length, got=%d want=%d", len(output), len(input))
	}
}

func TestAudioProcessor_ApplyAmbient_EmptyOnMixerError(t *testing.T) {
	audioProcessor := &AudioProcessor{
		ambientMixer: &twilioFakeMixer{err: errors.New("mix")},
	}

	if output := audioProcessor.applyAmbient(nil); output != nil {
		t.Fatalf("expected nil output for idle mix error, got=%v", output)
	}
}

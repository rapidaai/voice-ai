package internal_exotel

import (
	"errors"
	"testing"

	internal_ambient "github.com/rapidaai/api/assistant-api/internal/audio/ambient"
	"github.com/rapidaai/protos"
)

type exotelFakeResampler struct {
	out []byte
	err error
}

func (resampler *exotelFakeResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	if resampler.err != nil {
		return nil, resampler.err
	}
	return append([]byte(nil), resampler.out...), nil
}

func (resampler *exotelFakeResampler) Close() {}

type exotelFakeMixer struct {
	out []byte
	err error
}

func (mixer *exotelFakeMixer) Configure(_ internal_ambient.Config) error { return nil }

func (mixer *exotelFakeMixer) Mix(primary []byte) ([]byte, error) {
	if mixer.err != nil {
		return nil, mixer.err
	}
	if mixer.out != nil {
		return append([]byte(nil), mixer.out...), nil
	}
	return primary, nil
}

func (mixer *exotelFakeMixer) Reset() {}

func (mixer *exotelFakeMixer) CurrentConfig() internal_ambient.Config {
	return internal_ambient.Config{}
}

func TestAudioProcessor_ConvertOutputAudio_UsesResampler(t *testing.T) {
	audioProcessor := &AudioProcessor{
		resampler:        &exotelFakeResampler{out: []byte{7, 8}},
		downstreamConfig: &protos.AudioConfig{},
		exotelConfig:     &protos.AudioConfig{},
	}

	got, err := audioProcessor.convertOutputAudio([]byte{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != 7 {
		t.Fatalf("unexpected output: %v", got)
	}
}

func TestAudioProcessor_ApplyAmbient_FallbackOnError(t *testing.T) {
	input := []byte{1, 2, 3}
	audioProcessor := &AudioProcessor{
		ambientMixer: &exotelFakeMixer{err: errors.New("mix")},
	}

	output := audioProcessor.applyAmbient(input)
	if len(output) != len(input) {
		t.Fatalf("expected passthrough on mixer error")
	}
}

package ambient

import (
	"encoding/binary"
	"testing"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testResampler struct{}

func (resampler *testResampler) Resample(_ []byte, _, _ *protos.AudioConfig) ([]byte, error) {
	return []byte{0xE8, 0x03, 0x18, 0xFC}, nil
}

func (resampler *testResampler) Close() {}

func TestLoopMixer_MixDisabled_Passthrough(t *testing.T) {
	m, err := NewLoopMixer(MixerSpec{
		TargetAudioConfig: internal_audio.NewLinear16khzMonoAudioConfig(),
		FrameBytes:        4,
	})
	require.NoError(t, err)

	in := []byte{0x10, 0x00, 0x20, 0x00}
	out, err := m.Mix(in)
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestLoopMixer_MixAmbientOnly(t *testing.T) {
	m, err := NewLoopMixer(MixerSpec{
		TargetAudioConfig: internal_audio.NewLinear16khzMonoAudioConfig(),
		FrameBytes:        4,
	})
	require.NoError(t, err)

	m.cfg = NewConfig(ProfileCafe, 50)
	// Two LINEAR16 samples: 1000, -1000
	m.ambientPCM = make([]byte, 4)
	neg := int16(-1000)
	binary.LittleEndian.PutUint16(m.ambientPCM[0:2], uint16(int16(1000)))
	binary.LittleEndian.PutUint16(m.ambientPCM[2:4], uint16(neg))

	out, err := m.Mix(nil)
	require.NoError(t, err)
	require.Len(t, out, 4)
	assert.Equal(t, int16(500), int16(binary.LittleEndian.Uint16(out[0:2])))
	assert.Equal(t, int16(-500), int16(binary.LittleEndian.Uint16(out[2:4])))
}

func TestLoopMixer_ConfigureResamplesAmbientAsset(t *testing.T) {
	resampler := &testResampler{}
	m, err := NewLoopMixer(MixerSpec{
		Resampler:         resampler,
		TargetAudioConfig: internal_audio.NewLinear8khzMonoAudioConfig(),
		FrameBytes:        4,
	})
	require.NoError(t, err)

	require.NoError(t, m.Configure(NewConfig(ProfileCafe, 50)))

	require.Len(t, m.ambientPCM, 4)
}

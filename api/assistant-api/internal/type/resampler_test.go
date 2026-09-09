package internal_type

import "testing"

type testAudioStreamResampler struct{}

func (testAudioStreamResampler) Write([]byte) error {
	return nil
}

func (testAudioStreamResampler) Flush() error {
	return nil
}

func (testAudioStreamResampler) Close() {}

func TestAudioStreamResamplerContract(t *testing.T) {
	var writer AudioStreamResampler = testAudioStreamResampler{}

	if err := writer.Write([]byte{1, 2}); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	writer.Close()
}

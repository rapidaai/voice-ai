package internal_voice_rnnoise

/*
#cgo CFLAGS: -I.
#cgo LDFLAGS: -L. -lrnnoise
#include <rnnoise.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"unsafe"
)

// DenoiseState represents the RNNoise denoising state
type DenoiseState struct {
	cState *C.DenoiseState
	model  *RNNModel
}

// RNNModel represents a RNNoise neural network model
type RNNModel struct {
	cModel *C.RNNModel
}

// GetFrameSize returns the number of samples processed in a single frame
func GetFrameSize() int {
	return int(C.rnnoise_get_frame_size())
}

// Create initializes a new DenoiseState
// If model is nil, the default model is used
func Create(model *RNNModel) (*DenoiseState, error) {
	var cModel *C.RNNModel
	if model != nil {
		cModel = model.cModel
	}

	state := C.rnnoise_create(cModel)
	if state == nil {
		return nil, fmt.Errorf("failed to create RNNoise state")
	}

	return &DenoiseState{
		cState: state,
		model:  model,
	}, nil
}

// Destroy frees the DenoiseState resources
func (st *DenoiseState) Destroy() {
	if st.cState != nil {
		C.rnnoise_destroy(st.cState)
		st.cState = nil
	}
}

// ProcessFrame applies noise reduction to an audio frame
func (st *DenoiseState) ProcessFrame(input []float32) (float64, []float32, error) {
	if len(input) != GetFrameSize() {
		return 0, nil, fmt.Errorf("input must be %d samples", GetFrameSize())
	}
	output := make([]float32, GetFrameSize())
	cInput := (*C.float)(unsafe.Pointer(&input[0]))
	cOutput := (*C.float)(unsafe.Pointer(&output[0]))
	confidence := C.rnnoise_process_frame(st.cState, cOutput, cInput)
	return float64(confidence), output, nil
}

// Resample16kHzTo48kHz converts 16kHz 10ms frame to 48kHz 480 samples
func Resample16kHzTo48kHz(input []float32) []float32 {
	if len(input) != 160 { // 16kHz 10ms frame
		panic("input must be 160 samples (16kHz 10ms frame)")
	}

	output := make([]float32, 480)

	for i := 0; i < 480; i++ {
		// Map 16kHz to 48kHz with interpolation
		sourceIndex := float32(i) * 160.0 / 480.0
		lowerIndex := int(sourceIndex)
		upperIndex := lowerIndex + 1

		if upperIndex >= 160 {
			upperIndex = 159
		}

		// Linear interpolation
		fraction := sourceIndex - float32(lowerIndex)
		output[i] = input[lowerIndex]*(1-fraction) +
			input[upperIndex]*fraction
	}

	return output
}

// Downsample48kHzTo16kHz reduces 48kHz samples back to 16kHz
func Downsample48kHzTo16kHz(input []float32) []float32 {
	outputLen := len(input) / 3
	output := make([]float32, outputLen)

	for i := 0; i < outputLen; i++ {
		// Take every third sample
		output[i] = input[i*3]
	}

	return output
}

// ProcessFrame16kHz processes a 16kHz audio frame through RNNoise
func (st *DenoiseState) ProcessFrame16kHz(input []float32) (float64, []float32, error) {
	// Resample input to 48kHz
	resampled := Resample16kHzTo48kHz(input)

	// Process resampled frame
	confidence, denoised, err := st.ProcessFrame(resampled)
	if err != nil {
		return 0, nil, err
	}

	// Downsample back to 16kHz
	return confidence, Downsample48kHzTo16kHz(denoised), nil
}

// ModelFromBuffer creates a RNNModel from a byte slice
func ModelFromBuffer(buffer []byte) (*RNNModel, error) {
	if len(buffer) == 0 {
		return nil, fmt.Errorf("empty model buffer")
	}

	cModel := C.rnnoise_model_from_buffer(unsafe.Pointer(&buffer[0]), C.int(len(buffer)))
	if cModel == nil {
		return nil, fmt.Errorf("failed to create model from buffer")
	}

	return &RNNModel{cModel: cModel}, nil
}

// ModelFromFile creates a RNNModel from an open file
func ModelFromFile(file *os.File) (*RNNModel, error) {
	cFile := C.fdopen(C.int(file.Fd()), C.CString("rb"))
	if cFile == nil {
		return nil, fmt.Errorf("failed to open file")
	}
	defer C.fclose(cFile)

	cModel := C.rnnoise_model_from_file(cFile)
	if cModel == nil {
		return nil, fmt.Errorf("failed to create model from file")
	}

	return &RNNModel{cModel: cModel}, nil
}

// ModelFromFilename creates a RNNModel from a filename
func ModelFromFilename(filename string) (*RNNModel, error) {
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	cModel := C.rnnoise_model_from_filename(cFilename)
	if cModel == nil {
		return nil, fmt.Errorf("failed to create model from filename")
	}

	return &RNNModel{cModel: cModel}, nil
}

// Free releases resources associated with the RNNModel
func (m *RNNModel) Free() {
	if m.cModel != nil {
		C.rnnoise_model_free(m.cModel)
		m.cModel = nil
	}
}

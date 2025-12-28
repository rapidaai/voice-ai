// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_capturers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/rapidaai/pkg/commons"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type AudioSegment struct {
	Data      []byte
	Role      type_enums.MessageActor
	Timestamp time.Time
}

type audioCapturer struct {
	logger  commons.Logger
	storage storages.Storage
	//
	mutex         sync.Mutex     // Protects access to `conversation`
	captureDone   bool           // Flag to signal when capture is done
	conversation  []AudioSegment // Stores audio segments with timestamps
	SampleRate    int
	Channels      int
	BitsPerSample int
}

func (*audioCapturer) Name() string {
	return "aws-s3-audio-capturer"
}

func NewS3AudioCapturer(logger commons.Logger, opts *CapturerOptions) (AudioCapturer, error) {
	cfg, err := opts.S3Config(nil)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize s3 audio capturer")
	}
	cac := &audioCapturer{
		logger:        logger,
		storage:       storage_files.NewAwsFileStorage(cfg, logger),
		conversation:  make([]AudioSegment, 0),
		Channels:      1,
		BitsPerSample: 16,
		SampleRate:    24000,
	}
	return cac, nil
}

func (cac *audioCapturer) Capture(ctx context.Context, role type_enums.MessageActor, wavBytes []byte) error {
	cac.mutex.Lock()
	defer cac.mutex.Unlock()

	if len(wavBytes) == 0 {
		cac.logger.Warnf("Received empty audio data for role: %v", role)
		return nil
	}

	segment := AudioSegment{
		Data:      wavBytes,
		Role:      role,
		Timestamp: time.Now(),
	}

	cac.conversation = append(cac.conversation, segment)
	cac.captureDone = true
	return nil
}

func (cac *audioCapturer) CreateWAVHeader(dataSize int) []byte {
	var buf bytes.Buffer

	buf.Write([]byte("RIFF"))
	fileSize := uint32(36 + dataSize)
	binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.Write([]byte("WAVE"))

	buf.Write([]byte("fmt "))
	binary.Write(&buf, binary.LittleEndian, uint32(16))
	binary.Write(&buf, binary.LittleEndian, uint16(1))
	binary.Write(&buf, binary.LittleEndian, uint16(cac.Channels))
	binary.Write(&buf, binary.LittleEndian, uint32(cac.SampleRate))
	byteRate := cac.SampleRate * cac.Channels * cac.BitsPerSample / 8
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	blockAlign := cac.Channels * cac.BitsPerSample / 8
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&buf, binary.LittleEndian, uint16(cac.BitsPerSample))

	buf.Write([]byte("data"))
	binary.Write(&buf, binary.LittleEndian, uint32(dataSize))
	return buf.Bytes()
}

func (cac *audioCapturer) mergeAudioByRole(role type_enums.MessageActor) []byte {
	var mergedAudio []byte
	for _, segment := range cac.conversation {
		if segment.Role == role {
			mergedAudio = append(mergedAudio, segment.Data...)
		}
	}
	return mergedAudio
}

func (cac *audioCapturer) storeAudioFile(ctx context.Context, fileName string, audioData []byte) storages.StorageOutput {
	wavHeader := cac.CreateWAVHeader(len(audioData))
	completeAudio := append(wavHeader, audioData...)

	cac.logger.Debugf("Storing audio file: %s, size: %d bytes", fileName, len(completeAudio))
	return cac.storage.Store(ctx, fileName, completeAudio)
}

func (cac *audioCapturer) Persist(ctx context.Context, key string) (*CapturerOutput, error) {
	cac.mutex.Lock()
	defer cac.mutex.Unlock()
	cac.logger.Debugf("Persisting audio with key: %s in storage: %s", key, cac.storage.Name())
	if len(cac.conversation) == 0 {
		cac.logger.Warnf("No audio segments to persist for key: %s", key)
		return nil, fmt.Errorf("no audio segments to persist")
	}
	userAudio := cac.mergeAudioByRole(type_enums.UserActor)
	assistantAudio := cac.mergeAudioByRole(type_enums.AssistantActor)

	userFileName := fmt.Sprintf("%s/%d__user.wav", key, gorm_generator.ID())
	assistantFileName := fmt.Sprintf("%s/%d__assistant.wav", key, gorm_generator.ID())

	paths := make([]string, 0)
	userOutput := cac.storeAudioFile(ctx, userFileName, userAudio)
	cac.logger.Debugf("userOutput: %+v\n", userOutput)

	if userOutput.Error == nil {
		paths = append(paths, userOutput.CompletePath)
		cac.logger.Debugf("paths after user append: %v\n", paths)
	}

	assistantOutput := cac.storeAudioFile(ctx, assistantFileName, assistantAudio)
	cac.logger.Debugf("assistantOutput: %+v\n", assistantOutput)

	if assistantOutput.Error == nil {
		paths = append(paths, assistantOutput.CompletePath)
		cac.logger.Debugf("paths after assistant append: %v\n", paths)
	}
	ouput := &CapturerOutput{
		Paths: paths,
		Name:  cac.Name(),
	}
	cac.logger.Debugf("paths after assistant append: %v\n", ouput)
	return ouput, nil

}

package internal_capturer_factories

import (
	internal_capturers "github.com/rapidaai/api/assistant-api/internal/capturers"
	"github.com/rapidaai/pkg/commons"
)

type CapturerIdentifier string

const (
	AzureAudioCapturer  CapturerIdentifier = "azure-cloud-audio-capturer"
	GoogleAudioCapturer CapturerIdentifier = "google-cloud-audio-capturer"
	AWSS3AudioCapturer  CapturerIdentifier = "aws-s3-audio-capturer"
	AWSS3TextCapturer   CapturerIdentifier = "aws-s3-text-capturer"
)

func GetAudioCapturer(aa CapturerIdentifier, logger commons.Logger, opts *internal_capturers.CapturerOptions) (internal_capturers.AudioCapturer, error) {
	switch aa {
	default:
		return internal_capturers.NewS3AudioCapturer(logger, opts)
	}
}

func GetTextCapturer(aa CapturerIdentifier, logger commons.Logger, opts *internal_capturers.CapturerOptions) (internal_capturers.TextCapturer, error) {
	switch aa {
	default:
		return internal_capturers.NewS3TextCapturer(logger, opts)
	}
}

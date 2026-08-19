// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package deepgram_internal

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	interfaces "github.com/deepgram/deepgram-go-sdk/v3/pkg/client/interfaces"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

const (
	SpeechToTextTransformerName = "deepgram-stt"
	TextToSpeechTransformerName = "deepgram-tts"
	deepgramDefaultEndpoint     = "api.deepgram.com"
	STTAgentIDTag               = "agent:%d"
	STTConversationIDTag        = "conversation:%d"
)

const (
	STTDefaultModel          = "nova"
	STTDefaultLanguage       = "en-US"
	STTDefaultChannels       = 1
	STTDefaultSmartFormat    = true
	STTDefaultInterimResults = true
	STTDefaultFillerWords    = true
	STTDefaultVADEvents      = false
	STTDefaultEndpointing    = "5"
	STTDefaultPunctuate      = true
	STTDefaultNoDelay        = true
	STTDefaultSampleRate     = 16000
	STTDefaultDiarize        = false
	STTDefaultMultichannel   = false
)

const (
	IllegalVaultConfigErrorMessage = "illegal vault config"

	STTCredentialRequiredErrorMessage       = "deepgram-stt: credential is required"
	STTOnPacketRequiredErrorMessage         = "deepgram-stt: on packet handler is required"
	STTConnectionFailedErrorMessage         = "deepgram-stt: connection failed"
	STTConnectionNotInitializedErrorMessage = "deepgram-stt: connection is not initialized"
	STTFinalizeErrorMessage                 = "deepgram finalize error: %w"
	STTStreamErrorMessage                   = "deepgram stream error: %w"

	STTCredentialFailedLogMessage     = "deepgram-stt: Key from credential failed %+v"
	STTInitializationErrorLogMessage  = "deepgram-stt: error while initialization %s"
	STTConnectErrorLogMessage         = "deepgram-stt: error while performing connect"
	STTInitializationCompletedMessage = "deepgram-stt: initialization completed"
	STTFinalizeErrorLogMessage        = "deepgram-stt: error while finalizing deepgram utterance: %v"
	STTStreamErrorLogMessage          = "deepgram-stt: error while calling deepgram: %v"
	STTUnhandledEventLogMessage       = "UnhandledEvent %+v"

	STTInitializationFailureMetricDescription = "STT initialization failure count"
	STTConnectionFailureMetricDescription     = "STT connection failure count"
	STTInitializationLatencyMetricDescription = "STT initialization latency in milliseconds"
	STTFinalizeFailureMetricDescription       = "STT finalize failure count"
	STTStreamFailureMetricDescription         = "STT stream failure count"
	STTTimeToFirstTokenMetricDescription      = "STT time to first token from speech start in milliseconds"
	STTTimeToLastTokenMetricDescription       = "STT time to final token from speech start in milliseconds"
	STTLatencyMetricDescription               = "STT latency from speech end to final transcript in milliseconds"
	STTProviderErrorMetricDescription         = "STT provider error count"
)

type DeepgramTextToSpeechResponse struct {
	Type       string  `json:"type"`
	SequenceID float64 `json:"sequence_id,omitempty"`
	Code       string  `json:"code,omitempty"`
	Message    string  `json:"description,omitempty"`
}

type DeepgramOption struct {
	key      string
	endpoint string
	logger   commons.Logger
	mdlOpts  utils.Option
	tags     []string
}

func NewDeepgramOption(
	logger commons.Logger,
	vaultCredential *protos.VaultCredential,
	opts utils.Option,
	tags ...string,
) (*DeepgramOption, error) {
	raw := vaultCredential.GetValue().AsMap()
	cx, ok := raw["key"]
	if !ok {
		return nil, fmt.Errorf(IllegalVaultConfigErrorMessage)
	}
	key, ok := cx.(string)
	if !ok || strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf(IllegalVaultConfigErrorMessage)
	}
	endpoint := deepgramDefaultEndpoint
	if endpointValue, ok := raw["endpoint"]; ok {
		if endpointString, ok := endpointValue.(string); ok && endpointString != "" {
			endpoint = endpointString
		}
	}
	deepgramOption := &DeepgramOption{
		key:      key,
		endpoint: endpoint,
		logger:   logger,
		mdlOpts:  opts,
		tags:     append([]string(nil), tags...),
	}
	return deepgramOption, nil
}

func (dgOpt *DeepgramOption) GetEncoding() string {
	return "linear16"
}

func (dgOpt *DeepgramOption) GetKey() string {
	return dgOpt.key
}

func (dgOpt *DeepgramOption) GetEndpoint() string {
	return dgOpt.endpoint
}

func (dgOpt *DeepgramOption) ClientOptions() *interfaces.ClientOptions {
	return &interfaces.ClientOptions{
		APIKey:          dgOpt.GetKey(),
		Host:            dgOpt.GetEndpoint(),
		EnableKeepAlive: true,
	}
}

func (dgOpt *DeepgramOption) SpeechToTextOptions() *interfaces.LiveTranscriptionOptions {
	opts := &interfaces.LiveTranscriptionOptions{
		Model:          STTDefaultModel,
		Language:       STTDefaultLanguage,
		Channels:       STTDefaultChannels,
		SmartFormat:    STTDefaultSmartFormat,
		InterimResults: STTDefaultInterimResults,
		FillerWords:    STTDefaultFillerWords,
		VadEvents:      STTDefaultVADEvents,
		Endpointing:    STTDefaultEndpointing,
		Punctuate:      STTDefaultPunctuate,
		NoDelay:        STTDefaultNoDelay,
		Encoding:       dgOpt.GetEncoding(),
		SampleRate:     STTDefaultSampleRate,
		Diarize:        STTDefaultDiarize,
		Multichannel:   STTDefaultMultichannel,
		Tag:            make([]string, 0),
	}

	if language, err := dgOpt.mdlOpts.GetString(internal_options.ListenOptionLanguage); err == nil {
		opts.Language = language
	}

	if smartFormat, err := dgOpt.mdlOpts.GetBool(internal_options.ListenOptionSmartFormat); err == nil {
		opts.SmartFormat = smartFormat
	}

	if fillerWords, err := dgOpt.mdlOpts.GetBool(internal_options.ListenOptionFillerWords); err == nil {
		opts.FillerWords = fillerWords
	}
	if vadEvents, err := dgOpt.mdlOpts.GetBool(internal_options.ListenOptionVADEvents); err == nil {
		opts.VadEvents = vadEvents
	}
	if endpointing, err := dgOpt.mdlOpts.GetString(internal_options.ListenOptionEndpointing); err == nil {
		opts.Endpointing = endpointing
	}
	if punctuate, err := dgOpt.mdlOpts.GetBool(internal_options.ListenOptionPunctuate); err == nil {
		opts.Punctuate = punctuate
	}
	if diarize, err := dgOpt.mdlOpts.GetBool(internal_options.ListenOptionDiarize); err == nil {
		opts.Diarize = diarize
	}
	if multichannel, err := dgOpt.mdlOpts.GetBool(internal_options.ListenOptionMultichannel); err == nil {
		opts.Multichannel = multichannel
	}
	if model, err := dgOpt.mdlOpts.GetString(internal_options.ListenOptionModel); err == nil {
		opts.Model = model
	}
	opts.Tag = append(opts.Tag, dgOpt.tags...)

	if keywordsRaw, exists := dgOpt.mdlOpts[internal_options.ListenOptionKeyword]; exists {
		var keywords []string
		switch v := keywordsRaw.(type) {
		case string:
			trimmed := strings.Trim(v, "[]")
			keywords = strings.Fields(trimmed)
		case []interface{}:
			keywords = make([]string, len(v))
			for i, keyword := range v {
				if str, ok := keyword.(string); ok {
					keywords[i] = strings.TrimSpace(str)
				}
			}
		default:
			dgOpt.logger.Warnf("Unexpected type for keywords: %T", keywordsRaw)
		}
		if len(keywords) > 0 {
			if opts.Model == "nova-2" {
				opts.Keywords = keywords
			}
			if opts.Model == "nova-3" {
				opts.Keyterm = keywords
			}
		}
	}
	return opts
}

func (dgOpt *DeepgramOption) GetTextToSpeechConnectionString() string {
	params := url.Values{}
	params.Add("encoding", dgOpt.GetEncoding())
	params.Add("sample_rate", "16000")
	if model, err := dgOpt.mdlOpts.GetString(internal_options.SpeakOptionVoiceID); err == nil {
		params.Add("model", model)
	}
	return fmt.Sprintf("wss://%s/v1/speak?%s", dgOpt.GetEndpoint(), params.Encode())
}

type SttSessionMetrics struct {
	mu              sync.Mutex
	speechStartedAt time.Time
	speechEndedAt   time.Time
	ttftReported    bool
	ttltReported    bool
	latencyReported bool
}

func (metrics *SttSessionMetrics) ResetSpeech(speechStartedAt time.Time) {
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	metrics.speechStartedAt = speechStartedAt
	metrics.speechEndedAt = time.Time{}
	metrics.ttftReported = false
	metrics.ttltReported = false
	metrics.latencyReported = false
}

func (metrics *SttSessionMetrics) SetSpeechEndedAt(speechEndedAt time.Time) {
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	metrics.speechEndedAt = speechEndedAt
	metrics.latencyReported = false
}

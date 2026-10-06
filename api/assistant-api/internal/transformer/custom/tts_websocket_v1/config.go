// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_tts_websocket_v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	internal_transformer_custom_dsl "github.com/rapidaai/api/assistant-api/internal/transformer/custom/internal/dsl"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

const (
	defaultEncoding   = "LINEAR16"
	defaultSampleRate = 16000
)

const (
	credentialKeyBaseURLSnake = "base_url"
	credentialKeyBaseURLCamel = "baseUrl"
	credentialKeyHeaders      = "headers"
)

const (
	optionKeyVoiceID       = internal_options.SpeakOptionVoiceID
	optionKeyModel         = internal_options.SpeakOptionModel
	optionKeyLanguage      = internal_options.SpeakOptionLanguage
	optionKeyEncoding      = internal_options.SpeakOptionAudioEncoding
	optionKeySampleRate    = internal_options.SpeakOptionAudioSampleRate
	optionKeyQueryParams   = internal_options.SpeakOptionQueryParams
	optionKeyRequestRules  = internal_options.SpeakOptionRequestRules
	optionKeyResponseRules = internal_options.SpeakOptionResponseRules
)

const (
	frameTypeBinary = "binary"
	frameTypeJSON   = "json"
	frameTypeText   = "text"
)

const (
	requestPacketText      = "text"
	requestPacketDone      = "done"
	requestPacketInterrupt = "interrupt"
)

var queryContract = internal_transformer_custom_dsl.Contract{
	SupportedVariables: []string{
		"message_id",
		"voice_id",
		"model",
		"language",
		"encoding",
		"sample_rate",
	},
}

var requestRuleContract = internal_transformer_custom_dsl.Contract{
	SupportedRequestPackets: []string{
		requestPacketText,
		requestPacketDone,
		requestPacketInterrupt,
	},
	SupportedRequestFrames: []string{
		frameTypeBinary,
		frameTypeJSON,
		frameTypeText,
	},
	SupportedPathRoots: []string{
		"config",
		"packet",
	},
	RequestValidationScopes: map[string]any{
		requestPacketText: map[string]any{
			"config": map[string]any{
				"voice": map[string]any{
					"id": "voice_123",
				},
				"model":    "model_123",
				"language": "en-US",
				"audio": map[string]any{
					"encoding":    defaultEncoding,
					"sample_rate": defaultSampleRate,
				},
			},
			"packet": map[string]any{
				"kind":       requestPacketText,
				"message_id": "msg_123",
				"text":       "Hello world",
			},
		},
		requestPacketDone: map[string]any{
			"config": map[string]any{
				"voice": map[string]any{
					"id": "voice_123",
				},
				"model":    "model_123",
				"language": "en-US",
				"audio": map[string]any{
					"encoding":    defaultEncoding,
					"sample_rate": defaultSampleRate,
				},
			},
			"packet": map[string]any{
				"kind":       requestPacketDone,
				"message_id": "msg_123",
				"text":       "",
			},
		},
		requestPacketInterrupt: map[string]any{
			"config": map[string]any{
				"voice": map[string]any{
					"id": "voice_123",
				},
				"model":    "model_123",
				"language": "en-US",
				"audio": map[string]any{
					"encoding":    defaultEncoding,
					"sample_rate": defaultSampleRate,
				},
			},
			"packet": map[string]any{
				"kind":       requestPacketInterrupt,
				"message_id": "msg_123",
				"text":       "",
			},
		},
	},
}

var responseContract = internal_transformer_custom_dsl.Contract{
	SupportedResponseFrames: []string{
		frameTypeBinary,
		frameTypeJSON,
	},
	SupportedEmitKeys: []string{
		"audio",
		"message_id",
		"done",
		"error",
	},
	AllowedFrameSelectors: []string{
		frameTypeBinary,
	},
	AllowDecodeBase64: true,
}

type Config struct {
	BaseURL string
	Headers map[string]string

	VoiceID    string
	Model      string
	Language   string
	Encoding   string
	SampleRate int

	QueryParams   map[string]any
	RequestRules  []RequestRule
	ResponseRules []ResponseRule
}

type RequestRule = internal_transformer_custom_dsl.RequestRule
type RequestWhen = internal_transformer_custom_dsl.RequestWhen
type RequestSend = internal_transformer_custom_dsl.Send
type ResponseRule = internal_transformer_custom_dsl.ResponseRule
type ResponseWhen = internal_transformer_custom_dsl.When

type configParser struct {
	credential *protos.VaultCredential
	opts       utils.Option
}

func NewConfig(credential *protos.VaultCredential, opts utils.Option) (*Config, error) {
	parser := &configParser{credential: credential, opts: opts}
	config := &Config{
		Headers:     map[string]string{},
		QueryParams: map[string]any{},
		Encoding:    defaultEncoding,
		SampleRate:  defaultSampleRate,
	}

	if err := parser.loadCredential(config); err != nil {
		return nil, err
	}
	if err := parser.loadOptions(config); err != nil {
		return nil, err
	}
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (parser *configParser) loadCredential(config *Config) error {
	if parser.credential == nil || parser.credential.GetValue() == nil {
		return fmt.Errorf("custom-tts websocket_v1: base url must be specified in credentials")
	}

	raw := parser.credential.GetValue().AsMap()
	baseURLRaw, found := raw[credentialKeyBaseURLSnake]
	if !found {
		baseURLRaw, found = raw[credentialKeyBaseURLCamel]
	}
	if !found {
		return fmt.Errorf("custom-tts websocket_v1: base url must be specified in credentials")
	}

	baseURL, ok := baseURLRaw.(string)
	if !ok {
		return fmt.Errorf("custom-tts websocket_v1: base url must be a string")
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return fmt.Errorf("custom-tts websocket_v1: base url must not be empty")
	}
	config.BaseURL = baseURL

	if rawHeaders, found := raw[credentialKeyHeaders]; found && rawHeaders != nil {
		headers, err := utils.Option{credentialKeyHeaders: rawHeaders}.GetStringMap(credentialKeyHeaders)
		if err != nil {
			return fmt.Errorf("custom-tts websocket_v1: invalid headers: %w", err)
		}
		if headers != nil {
			config.Headers = headers
		}
	}

	return nil
}

func (parser *configParser) loadOptions(config *Config) error {
	if voiceID, err := parser.opts.GetString(optionKeyVoiceID); err == nil {
		config.VoiceID = strings.TrimSpace(voiceID)
	}
	if model, err := parser.opts.GetString(optionKeyModel); err == nil {
		config.Model = strings.TrimSpace(model)
	}
	if language, err := parser.opts.GetString(optionKeyLanguage); err == nil {
		config.Language = strings.TrimSpace(language)
	}
	if encoding, err := parser.opts.GetString(optionKeyEncoding); err == nil && strings.TrimSpace(encoding) != "" {
		config.Encoding = strings.TrimSpace(encoding)
	}
	if rawSampleRate, found := parser.opts[optionKeySampleRate]; found && rawSampleRate != nil {
		sampleRate, err := parser.opts.GetUint32(optionKeySampleRate)
		if err != nil {
			return fmt.Errorf("custom-tts websocket_v1: invalid %s: %w", optionKeySampleRate, err)
		}
		config.SampleRate, err = utils.Uint32ToInt(sampleRate)
		if err != nil {
			return fmt.Errorf("custom-tts websocket_v1: invalid %s: %w", optionKeySampleRate, err)
		}
	}

	if found, err := parser.decodeJSONObject(optionKeyQueryParams, false, &config.QueryParams); err != nil {
		return err
	} else if found && config.QueryParams == nil {
		config.QueryParams = map[string]any{}
	}
	if _, err := parser.decodeJSONArray(optionKeyRequestRules, true, &config.RequestRules); err != nil {
		return err
	}
	if _, err := parser.decodeJSONArray(optionKeyResponseRules, true, &config.ResponseRules); err != nil {
		return err
	}

	return nil
}

func (parser *configParser) decodeJSONObject(key string, required bool, destination *map[string]any) (bool, error) {
	raw, found := parser.opts[key]
	if !found || raw == nil {
		if required {
			return false, fmt.Errorf("custom-tts websocket_v1: %s is required", key)
		}
		return false, nil
	}
	if !required {
		rawString, ok := raw.(string)
		if !ok || !validator.NotBlank(rawString) {
			return false, nil
		}
	}

	payload, err := parser.toJSONBytes(raw, key)
	if err != nil {
		return true, err
	}
	if err := parser.decodeJSON(payload, destination, key); err != nil {
		return true, err
	}
	return true, nil
}

func (parser *configParser) decodeJSONArray(key string, required bool, destination any) (bool, error) {
	raw, found := parser.opts[key]
	if !found || raw == nil {
		if required {
			return false, fmt.Errorf("custom-tts websocket_v1: %s is required", key)
		}
		return false, nil
	}

	payload, err := parser.toJSONBytes(raw, key)
	if err != nil {
		return true, err
	}
	if err := parser.decodeJSON(payload, destination, key); err != nil {
		return true, err
	}
	return true, nil
}

func (parser *configParser) toJSONBytes(raw any, key string) ([]byte, error) {
	switch typed := raw.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil, fmt.Errorf("custom-tts websocket_v1: invalid %s: value must not be empty", key)
		}
		return []byte(trimmed), nil
	case []byte:
		trimmed := bytes.TrimSpace(typed)
		if len(trimmed) == 0 {
			return nil, fmt.Errorf("custom-tts websocket_v1: invalid %s: value must not be empty", key)
		}
		return trimmed, nil
	default:
		payload, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("custom-tts websocket_v1: invalid %s: %w", key, err)
		}
		return payload, nil
	}
}

func (parser *configParser) decodeJSON(payload []byte, destination any, key string) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("custom-tts websocket_v1: invalid %s: %w", key, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("custom-tts websocket_v1: invalid %s: trailing content after JSON value", key)
		}
		return fmt.Errorf("custom-tts websocket_v1: invalid %s: trailing content after JSON value: %w", key, err)
	}
	return nil
}

func (config *Config) validate() error {
	core := internal_transformer_custom_dsl.NewCore("custom-tts websocket_v1")
	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("custom-tts websocket_v1: base url must be specified in credentials")
	}
	if strings.TrimSpace(config.Encoding) == "" {
		return fmt.Errorf("custom-tts websocket_v1: %s must not be empty", optionKeyEncoding)
	}
	if config.SampleRate <= 0 {
		return fmt.Errorf("custom-tts websocket_v1: %s must be positive", optionKeySampleRate)
	}
	if len(config.RequestRules) == 0 {
		return fmt.Errorf("custom-tts websocket_v1: %s must contain at least one rule", optionKeyRequestRules)
	}
	if len(config.ResponseRules) == 0 {
		return fmt.Errorf("custom-tts websocket_v1: %s must contain at least one rule", optionKeyResponseRules)
	}

	if len(config.QueryParams) > 0 {
		if err := core.ValidateQueryParams(config.QueryParams, queryContract, optionKeyQueryParams); err != nil {
			return err
		}
	}
	if err := core.ValidateRequestRules(config.RequestRules, requestRuleContract, optionKeyRequestRules); err != nil {
		return err
	}
	hasTextRule := false
	for _, rule := range config.RequestRules {
		if strings.TrimSpace(rule.When.Packet) == requestPacketText {
			hasTextRule = true
			break
		}
	}
	if !hasTextRule {
		return fmt.Errorf(
			"custom-tts websocket_v1: %s must contain at least one rule with when.packet %q",
			optionKeyRequestRules,
			requestPacketText,
		)
	}
	if err := core.ValidateResponseRules(config.ResponseRules, responseContract, optionKeyResponseRules); err != nil {
		return err
	}

	return nil
}

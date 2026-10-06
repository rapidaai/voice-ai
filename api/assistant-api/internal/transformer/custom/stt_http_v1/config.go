// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_transformer_custom_stt_http_v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
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
	optionKeyModel         = internal_options.ListenOptionModel
	optionKeyLanguage      = internal_options.ListenOptionLanguage
	optionKeyEncoding      = internal_options.ListenOptionAudioEncoding
	optionKeySampleRate    = internal_options.ListenOptionAudioSampleRate
	optionKeyQueryParams   = internal_options.ListenOptionQueryParams
	optionKeyRequestRules  = internal_options.ListenOptionRequestRules
	optionKeyResponseRules = internal_options.ListenOptionResponseRules
)

const (
	frameTypeJSON   = "json"
	frameTypeBinary = "binary"
	frameTypeText   = "text"
)

const (
	requestPacketTurnChange = "turn_change"
	requestPacketAudio      = "audio"
	requestPacketInterrupt  = "interrupt"
)

var queryContract = internal_transformer_custom_dsl.Contract{
	SupportedVariables: []string{
		"model",
		"language",
		"encoding",
		"sample_rate",
	},
}

var requestRuleContract = internal_transformer_custom_dsl.Contract{
	SupportedRequestPackets: []string{
		requestPacketTurnChange,
		requestPacketAudio,
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
		requestPacketTurnChange: map[string]any{
			"config": map[string]any{
				"model":    "model-a",
				"language": "en-US",
				"audio": map[string]any{
					"encoding":    defaultEncoding,
					"sample_rate": defaultSampleRate,
				},
			},
			"packet": map[string]any{
				"kind":       requestPacketTurnChange,
				"context_id": "ctx_123",
			},
		},
		requestPacketAudio: map[string]any{
			"config": map[string]any{
				"model":    "model-a",
				"language": "en-US",
				"audio": map[string]any{
					"encoding":    defaultEncoding,
					"sample_rate": defaultSampleRate,
				},
			},
			"packet": map[string]any{
				"kind":       requestPacketAudio,
				"context_id": "ctx_123",
				"audio": map[string]any{
					"bytes":      []byte{0x00, 0x01},
					"base64":     "AAE=",
					"pcm_base64": "AAE=",
					"wav_base64": "UklGRiYAAABXQVZFZm10IBAAAAABAAEAgD4AAAB9AAACABAAZGF0YQIAAAAAAQ==",
				},
			},
		},
		requestPacketInterrupt: map[string]any{
			"config": map[string]any{
				"model":    "model-a",
				"language": "en-US",
				"audio": map[string]any{
					"encoding":    defaultEncoding,
					"sample_rate": defaultSampleRate,
				},
			},
			"packet": map[string]any{
				"kind":       requestPacketInterrupt,
				"context_id": "ctx_123",
			},
		},
	},
}

var responseContract = internal_transformer_custom_dsl.Contract{
	SupportedResponseFrames: []string{
		frameTypeJSON,
		frameTypeText,
	},
	SupportedEmitKeys: []string{
		"script",
		"confidence",
		"language",
		"interim",
		"error",
	},
	AllowedFrameSelectors: []string{
		frameTypeText,
	},
}

type Config struct {
	BaseURL string
	Headers map[string]string

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
		return fmt.Errorf("custom-stt http_v1: base url must be specified in credentials")
	}

	raw := parser.credential.GetValue().AsMap()
	baseURLRaw, found := raw[credentialKeyBaseURLSnake]
	if !found {
		baseURLRaw, found = raw[credentialKeyBaseURLCamel]
	}
	if !found {
		return fmt.Errorf("custom-stt http_v1: base url must be specified in credentials")
	}

	baseURL, ok := baseURLRaw.(string)
	if !ok {
		return fmt.Errorf("custom-stt http_v1: base url must be a string")
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return fmt.Errorf("custom-stt http_v1: base url must not be empty")
	}
	config.BaseURL = baseURL

	if rawHeaders, found := raw[credentialKeyHeaders]; found && rawHeaders != nil {
		headers, err := utils.Option{credentialKeyHeaders: rawHeaders}.GetStringMap(credentialKeyHeaders)
		if err != nil {
			return fmt.Errorf("custom-stt http_v1: invalid headers: %w", err)
		}
		if headers != nil {
			config.Headers = headers
		}
	}

	return nil
}

func (parser *configParser) loadOptions(config *Config) error {
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
			return fmt.Errorf("custom-stt http_v1: invalid %s: %w", optionKeySampleRate, err)
		}
		config.SampleRate, err = utils.Uint32ToInt(sampleRate)
		if err != nil {
			return fmt.Errorf("custom-stt http_v1: invalid %s: %w", optionKeySampleRate, err)
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
			return false, fmt.Errorf("custom-stt http_v1: %s is required", key)
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
			return false, fmt.Errorf("custom-stt http_v1: %s is required", key)
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
			return nil, fmt.Errorf("custom-stt http_v1: invalid %s: value must not be empty", key)
		}
		return []byte(trimmed), nil
	case []byte:
		trimmed := bytes.TrimSpace(typed)
		if len(trimmed) == 0 {
			return nil, fmt.Errorf("custom-stt http_v1: invalid %s: value must not be empty", key)
		}
		return trimmed, nil
	default:
		payload, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("custom-stt http_v1: invalid %s: %w", key, err)
		}
		return payload, nil
	}
}

func (parser *configParser) decodeJSON(payload []byte, destination any, key string) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("custom-stt http_v1: invalid %s: %w", key, err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("custom-stt http_v1: invalid %s: trailing content after JSON value", key)
		}
		return fmt.Errorf("custom-stt http_v1: invalid %s: trailing content after JSON value: %w", key, err)
	}
	return nil
}

func (config *Config) validate() error {
	core := internal_transformer_custom_dsl.NewCore("custom-stt http_v1")
	if strings.TrimSpace(config.BaseURL) == "" {
		return fmt.Errorf("custom-stt http_v1: base url must be specified in credentials")
	}
	if strings.TrimSpace(config.Encoding) == "" {
		return fmt.Errorf("custom-stt http_v1: %s must not be empty", optionKeyEncoding)
	}
	if !validator.Between(config.SampleRate, 1, math.MaxInt) {
		return fmt.Errorf("custom-stt http_v1: %s must be positive", optionKeySampleRate)
	}
	if len(config.RequestRules) == 0 {
		return fmt.Errorf("custom-stt http_v1: %s must contain at least one rule", optionKeyRequestRules)
	}
	if len(config.ResponseRules) == 0 {
		return fmt.Errorf("custom-stt http_v1: %s must contain at least one rule", optionKeyResponseRules)
	}

	if len(config.QueryParams) > 0 {
		if err := core.ValidateQueryParams(config.QueryParams, queryContract, optionKeyQueryParams); err != nil {
			return err
		}
	}
	if err := core.ValidateRequestRules(config.RequestRules, requestRuleContract, optionKeyRequestRules); err != nil {
		return err
	}
	hasAudioRule := false
	for _, rule := range config.RequestRules {
		if strings.TrimSpace(rule.When.Packet) == requestPacketAudio {
			hasAudioRule = true
			break
		}
	}
	if !hasAudioRule {
		return fmt.Errorf(
			"custom-stt http_v1: %s must contain at least one rule with when.packet %q",
			optionKeyRequestRules,
			requestPacketAudio,
		)
	}
	if err := core.ValidateResponseRules(config.ResponseRules, responseContract, optionKeyResponseRules); err != nil {
		return err
	}

	return nil
}

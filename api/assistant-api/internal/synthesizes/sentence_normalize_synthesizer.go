// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_synthesizers

import (
	"context"
	"errors"
	"strings"

	internal_normalizers "github.com/rapidaai/api/assistant-api/internal/synthesizes/normalizers"
	"github.com/rapidaai/pkg/commons"
)

type sentenceNormalizeSynthesizer struct {
	logger      commons.Logger
	normalizers []internal_normalizers.Normalizer
}

var normalizerMap = map[string]func(commons.Logger) internal_normalizers.Normalizer{
	"currency":             internal_normalizers.NewCurrencyNormalizer,
	"date":                 internal_normalizers.NewDateNormalizer,
	"time":                 internal_normalizers.NewTimeNormalizer,
	"numeral":              internal_normalizers.NewNumberToWordNormalizer,
	"address":              internal_normalizers.NewAddressNormalizer,
	"url":                  internal_normalizers.NewUrlNormalizer,
	"tech-abbreviation":    internal_normalizers.NewTechAbbreviationNormalizer,
	"role-abbreviation":    internal_normalizers.NewRoleAbbreviationNormalizer,
	"general-abbreviation": internal_normalizers.NewGeneralAbbreviationNormalizer,
	"symbol":               internal_normalizers.NewSymbolNormalizer,
}

func NewSentenceNormalizeSynthesizer(logger commons.Logger, opts SynthesizerOptions) (SentenceSynthesizer, error) {
	dictionariesInterface, err := opts.SpeakerOptions.GetString("speaker.pronunciation.dictionaries")
	if err != nil {
		return nil, errors.New("no synthesizer applied")
	}
	dictionaries := strings.Split(dictionariesInterface, commons.SEPARATOR)
	normalizers := make([]internal_normalizers.Normalizer, 0, len(dictionaries))
	for _, dict := range dictionaries {
		if normalizerFunc, ok := normalizerMap[strings.TrimSpace(dict)]; ok {
			normalizers = append(normalizers, normalizerFunc(logger))
		}
	}
	return &sentenceNormalizeSynthesizer{
		logger:      logger,
		normalizers: normalizers,
	}, nil
}

func (ess *sentenceNormalizeSynthesizer) Synthesize(
	ctx context.Context,
	contextId string,
	text string,
) string {
	for _, v := range ess.normalizers {
		text = v.Normalize(text)
	}
	return text
}

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_normalizers

import (
	"regexp"

	"github.com/rapidaai/pkg/commons"
)

type addressNormalizer struct {
	logger       commons.Logger
	replacements map[string]string
}

func NewAddressNormalizer(logger commons.Logger) Normalizer {
	return &addressNormalizer{
		logger: logger,
		replacements: map[string]string{
			`(?i)\bst\b`:   "street",
			`(?i)\bave\b`:  "avenue",
			`(?i)\brd\b`:   "road",
			`(?i)\bblvd\b`: "boulevard",
		},
	}
}

func (an *addressNormalizer) Normalize(s string) string {
	for abbr, full := range an.replacements {
		re := regexp.MustCompile(abbr)
		s = re.ReplaceAllString(s, full)
	}
	return s
}

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_output_processors

import (
	"regexp"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

type addressProcessor struct {
	logger       commons.Logger
	replacements map[string]string
}

func NewAddressProcessor(logger commons.Logger) internal_type.TextProcessor {
	return &addressProcessor{
		logger: logger,
		replacements: map[string]string{
			`(?i)\bst\b`:   "street",
			`(?i)\bave\b`:  "avenue",
			`(?i)\brd\b`:   "road",
			`(?i)\bblvd\b`: "boulevard",
		},
	}
}

func (an *addressProcessor) Process(s string) string {
	for abbr, full := range an.replacements {
		re := regexp.MustCompile(abbr)
		s = re.ReplaceAllString(s, full)
	}
	return s
}

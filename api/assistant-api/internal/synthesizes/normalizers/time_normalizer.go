// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_normalizers

import (
	"regexp"
	"time"

	"github.com/rapidaai/pkg/commons"
)

type timeNormalizer struct {
	logger commons.Logger
	re     *regexp.Regexp
}

func NewTimeNormalizer(logger commons.Logger) Normalizer {
	return &timeNormalizer{
		logger: logger,
		re:     regexp.MustCompile(`(\d{1,2}):(\d{2})`),
	}
}

func (tn *timeNormalizer) Normalize(s string) string {
	return tn.re.ReplaceAllStringFunc(s, func(match string) string {
		t, err := time.Parse("15:04", match)
		if err != nil {
			tn.logger.Warn("Failed to parse time", "error", err, "time", match)
			return match
		}
		return t.Format("3:04 PM")
	})
}

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_output_processors

import (
	"regexp"
	"strings"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

type urlProcessor struct {
	logger commons.Logger
}

func NewUrlProcessor(logger commons.Logger) internal_type.TextProcessor {
	return &urlProcessor{
		logger: logger,
	}
}

func (un *urlProcessor) Process(s string) string {
	re := regexp.MustCompile(`(https?://)?([^\s.]+\.[^\s]{2,}|www\.[^\s]+\.[^\s]{2,})`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		return strings.ReplaceAll(match, ".", " dot ")
	})
}

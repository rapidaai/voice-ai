// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_output_processors

import (
	"regexp"
	"time"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

type dateProcessor struct {
	logger commons.Logger
	re     *regexp.Regexp
}

func NewDateProcessor(logger commons.Logger) internal_type.TextProcessor {
	return &dateProcessor{
		logger: logger,
		re: regexp.MustCompile(
			`(\d{4}-\d{2}-\d{2})|` + // YYYY-MM-DD
				`(\d{2}/\d{2}/\d{4})|` + // DD/MM/YYYY or MM/DD/YYYY
				`(\d{2}-\d{2}-\d{4})|` + // DD-MM-YYYY
				`(\d{4}\.\d{2}\.\d{2})`, // YYYY.MM.DD
		),
	}
}

func (dn *dateProcessor) Process(s string) string {
	return dn.re.ReplaceAllStringFunc(s, func(match string) string {
		var date time.Time
		var err error

		formats := []string{
			"2006-01-02", // YYYY-MM-DD
			"02/01/2006", // DD/MM/YYYY
			"01/02/2006", // MM/DD/YYYY
			"02-01-2006", // DD-MM-YYYY
			"2006.01.02", // YYYY.MM.DD
		}

		for _, format := range formats {
			date, err = time.Parse(format, match)
			if err == nil {
				break
			}
		}

		if err != nil {
			dn.logger.Warn("Failed to parse date", "error", err, "date", match)
			return match
		}
		return date.Format("January 2, 2006")
	})
}

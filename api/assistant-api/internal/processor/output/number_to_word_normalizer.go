// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_output_processors

import (
	"regexp"
	"strconv"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

type numberToWordProcessor struct {
	logger commons.Logger
	re     *regexp.Regexp
}

func NewNumberToWordProcessor(logger commons.Logger) internal_type.TextProcessor {
	return &numberToWordProcessor{
		logger: logger,
		re:     regexp.MustCompile(`\b\d{1,2}\b`),
	}
}

func (nwn *numberToWordProcessor) Process(s string) string {
	return nwn.re.ReplaceAllStringFunc(s, func(match string) string {
		num, err := strconv.Atoi(match)
		if err != nil {
			nwn.logger.Warn("Failed to parse number", "error", err, "number", match)
			return match
		}
		return nwn.numberToWord(num)
	})
}

func (nwn *numberToWordProcessor) numberToWord(num int) string {
	if num < 0 || num > 99 {
		return strconv.Itoa(num)
	}

	units := []string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}
	teens := []string{"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	tens := []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

	if num < 10 {
		return units[num]
	} else if num < 20 {
		// #nosec G602, num is checked to be between 10 and 19.
		return teens[num-10]
	} else {
		ten := num / 10
		unit := num % 10
		if unit == 0 {
			return tens[ten]
		}
		return tens[ten] + "-" + units[unit]
	}
}

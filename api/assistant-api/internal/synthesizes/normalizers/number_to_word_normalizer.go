// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_normalizers

import (
	"regexp"
	"strconv"

	"github.com/rapidaai/pkg/commons"
)

type numberToWordNormalizer struct {
	logger commons.Logger
	re     *regexp.Regexp
}

func NewNumberToWordNormalizer(logger commons.Logger) Normalizer {
	return &numberToWordNormalizer{
		logger: logger,
		re:     regexp.MustCompile(`\b\d{1,2}\b`),
	}
}

func (nwn *numberToWordNormalizer) Normalize(s string) string {
	return nwn.re.ReplaceAllStringFunc(s, func(match string) string {
		num, err := strconv.Atoi(match)
		if err != nil {
			nwn.logger.Warn("Failed to parse number", "error", err, "number", match)
			return match
		}
		return nwn.numberToWord(num)
	})
}

func (nwn *numberToWordNormalizer) numberToWord(num int) string {
	if num < 0 || num > 99 {
		return strconv.Itoa(num)
	}

	units := []string{"", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}
	teens := []string{"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
	tens := []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

	if num < 10 {
		return units[num]
	} else if num < 20 {
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

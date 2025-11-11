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

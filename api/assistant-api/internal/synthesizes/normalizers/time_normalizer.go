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

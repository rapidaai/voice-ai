package internal_normalizers

import (
	"regexp"
	"strings"

	"github.com/rapidaai/pkg/commons"
)

type urlNormalizer struct {
	logger commons.Logger
}

func NewUrlNormalizer(logger commons.Logger) Normalizer {
	return &urlNormalizer{
		logger: logger,
	}
}

func (un *urlNormalizer) Normalize(s string) string {
	re := regexp.MustCompile(`(https?://)?([^\s.]+\.[^\s]{2,}|www\.[^\s]+\.[^\s]{2,})`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		return strings.ReplaceAll(match, ".", " dot ")
	})
}

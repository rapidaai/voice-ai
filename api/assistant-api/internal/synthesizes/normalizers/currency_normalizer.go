package internal_normalizers

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/rapidaai/pkg/commons"
	ntw "moul.io/number-to-words"
)

type currencyNormalizer struct {
	logger commons.Logger
	re     *regexp.Regexp
}

func NewCurrencyNormalizer(logger commons.Logger) Normalizer {
	return &currencyNormalizer{
		logger: logger,
		re:     regexp.MustCompile(`\$([0-9,]+)\.(\d{2})`),
	}
}

func (cn *currencyNormalizer) Normalize(s string) string {
	return cn.re.ReplaceAllStringFunc(s, func(match string) string {
		parts := cn.re.FindStringSubmatch(match)
		dollarStr := strings.ReplaceAll(parts[1], ",", "")
		dollarAmount, err := strconv.Atoi(dollarStr)
		if err != nil {
			cn.logger.Warn("Failed to parse dollar amount", "error", err, "amount", parts[1])
			return match
		}
		centAmount, err := strconv.Atoi(parts[2])
		if err != nil {
			cn.logger.Warn("Failed to parse cent amount", "error", err, "amount", parts[2])
			return match
		}

		dollars := ntw.IntegerToEnUs(dollarAmount)
		cents := ntw.IntegerToEnUs(centAmount)

		return dollars + " dollars and " + cents + " cents"
	})
}

package internal_synthesizers

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/rapidaai/pkg/commons"
)

type sentenceFormattingSynthesizer struct {
	logger commons.Logger
	opts   SynthesizerOptions
}

func NewSentenceFormattingSynthesizer(logger commons.Logger, opts SynthesizerOptions) (SentenceSynthesizer, error) {
	return &sentenceFormattingSynthesizer{
		logger: logger,
		opts:   opts,
	}, nil
}

func (ess *sentenceFormattingSynthesizer) Synthesize(
	ctx context.Context,
	contextId string,
	text string,
) string {
	text = ess.RemoveMarkdown(text)
	return ess.AddConjectionPause(text)
}

func (ess *sentenceFormattingSynthesizer) AddConjectionPause(text string) string {
	conjuctionBoundaries, err := ess.opts.SpeakerOptions.GetString("speaker.conjunction.boundaries")
	if err != nil {
		return text
	}

	conjunctions := strings.Split(conjuctionBoundaries, commons.SEPARATOR)
	for _, conj := range conjunctions {
		pattern := fmt.Sprintf(`\s+(%s)\s+`, conj)
		replacement := ` - $1 `
		re := regexp.MustCompile(pattern)
		text = re.ReplaceAllString(text, replacement)
	}
	return text
}

func (ess *sentenceFormattingSynthesizer) RemoveMarkdown(input string) string {
	// Remove headers (#, ##, ### ...)
	re := regexp.MustCompile(`(?m)^#{1,6}\s*`)
	output := re.ReplaceAllString(input, "")

	// Remove bold/italic markers (*, **, _, __)
	re = regexp.MustCompile(`\*{1,2}([^*]+?)\*{1,2}|_{1,2}([^_]+?)_{1,2}`)
	output = re.ReplaceAllString(output, "$1$2")

	// Remove inline code/backticks
	re = regexp.MustCompile("`([^`]+)`")
	output = re.ReplaceAllString(output, "$1")

	// Remove blockquotes (>)
	re = regexp.MustCompile(`(?m)^>\s?`)
	output = re.ReplaceAllString(output, "")

	// Remove links [text](url) → keep text
	re = regexp.MustCompile(`\[(.*?)\]\(.*?\)`)
	output = re.ReplaceAllString(output, "$1")

	// Remove images ![alt](url) → keep alt
	re = regexp.MustCompile(`!\[(.*?)\]\(.*?\)`)
	output = re.ReplaceAllString(output, "$1")

	// Remove horizontal rules (---, ***)
	re = regexp.MustCompile(`(?m)^(-{3,}|\*{3,}|_{3,})$`)
	output = re.ReplaceAllString(output, "")

	// Remove extra asterisks/underscores
	re = regexp.MustCompile(`[*_]+`)
	output = re.ReplaceAllString(output, "")

	return output
}

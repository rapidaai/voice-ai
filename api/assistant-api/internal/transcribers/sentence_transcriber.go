package internal_transcribes

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/rapidaai/pkg/commons"
)

type sentenceTranscriber struct {
	options          TranscriberOptions
	buffer           strings.Builder
	currentContext   string
	logger           commons.Logger
	mu               sync.RWMutex
	boundaryRegex    *regexp.Regexp
	customBoundaries []string
	hasBoundaries    bool
}

func NewSentenceTranscriber(logger commons.Logger, options TranscriberOptions) (Transcriber, error) {
	st := &sentenceTranscriber{
		options: options,
		logger:  logger,
	}
	if boundariesRaw, err := st.options.Opts.GetString("speaker.sentence.boundaries"); err == nil {
		if boundariesRaw != "" {
			boundaries := strings.Split(boundariesRaw, commons.SEPARATOR)
			var validBoundaries []string
			for _, b := range boundaries {
				if b = strings.TrimSpace(b); b != "" {
					validBoundaries = append(validBoundaries, b)
				}
			}

			if len(validBoundaries) > 0 {
				st.customBoundaries = validBoundaries
				st.hasBoundaries = true
				var parts []string
				for _, b := range validBoundaries {
					parts = append(parts, regexp.QuoteMeta(b))
				}
				pattern := fmt.Sprintf(`(%s)\s*`, strings.Join(parts, "|"))
				var err error
				st.boundaryRegex, err = regexp.Compile(pattern)
				if err != nil {
					logger.Errorf("Invalid boundary regex: %v", err)
					st.boundaryRegex = nil
					st.hasBoundaries = false
				} else {
					logger.Debugf("Custom sentence boundaries: %v", validBoundaries)
				}
			}
		}
	}
	if !st.hasBoundaries {
		logger.Debug("No sentence boundaries defined — will emit only on completion")
	}
	return st, nil
}

func (st *sentenceTranscriber) Transcribe(ctx context.Context, contextId string, text string, completed bool) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if contextId != st.currentContext && st.currentContext != "" {
		if err := st.flushUnsafe(ctx, st.currentContext); err != nil {
			st.logger.Errorf("Failed to flush context %s: %v", st.currentContext, err)
			return err
		}
	}
	st.currentContext = contextId
	st.buffer.WriteString(text)
	if st.hasBoundaries {
		for {
			sentence, remaining := st.extractSentence(st.buffer.String())
			if sentence == "" {
				break
			}
			if sentence != "" {
				if err := st.options.OnCompleteSentence(ctx, contextId, sentence); err != nil {
					return err
				}
			}
			st.buffer.Reset()
			st.buffer.WriteString(remaining)
		}
	}
	if completed {
		return st.flushUnsafe(ctx, contextId)
	}
	return nil
}

func (st *sentenceTranscriber) extractSentence(text string) (string, string) {
	if st.boundaryRegex == nil || text == "" {
		return "", text
	}
	loc := st.boundaryRegex.FindStringIndex(text)
	if loc != nil {
		return strings.TrimSpace(text[:loc[1]]), text[loc[1]:]
	}
	return "", text
}

func (st *sentenceTranscriber) Flush(ctx context.Context, contextId string) error {
	if ctx == nil {
		return errors.New("context cannot be nil")
	}
	if contextId == "" {
		return errors.New("contextId cannot be empty")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.flushUnsafe(ctx, contextId)
}

func (st *sentenceTranscriber) flushUnsafe(ctx context.Context, contextId string) error {
	remaining := strings.TrimSpace(st.buffer.String())
	if remaining == "" {
		st.buffer.Reset()
		return nil
	}
	if remaining != "" {
		if err := st.options.OnCompleteSentence(ctx, contextId, remaining); err != nil {
			return err
		}
	}
	st.buffer.Reset()
	return nil
}

func (st *sentenceTranscriber) GetCurrentBuffer() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.buffer.String()
}

func (st *sentenceTranscriber) GetCurrentContext() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.currentContext
}

func (st *sentenceTranscriber) Reset() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.buffer.Reset()
	st.currentContext = ""
}

func (st *sentenceTranscriber) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.buffer.Reset()
	st.currentContext = ""
	return nil
}

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_tokenizer

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type sentenceTokenizer struct {
	mu       sync.RWMutex
	logger   commons.Logger
	callback TokenizerCallback
	options  utils.Option

	buffer           strings.Builder
	currentContext   string
	boundaryRegex    *regexp.Regexp
	customBoundaries []string
	hasBoundaries    bool
}

func NewSentenceTokenizer(logger commons.Logger, callback TokenizerCallback, options utils.Option) (Tokenizer, error) {
	st := &sentenceTokenizer{
		options:  options,
		callback: callback,
		logger:   logger,
	}
	if boundariesRaw, err := st.options.GetString("speaker.sentence.boundaries"); err == nil {
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

func (st *sentenceTokenizer) Tokenize(ctx context.Context, contextId string, text string, completed bool) error {
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
				if err := st.callback(ctx, contextId, sentence); err != nil {
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

func (st *sentenceTokenizer) extractSentence(text string) (string, string) {
	if st.boundaryRegex == nil || text == "" {
		return "", text
	}
	loc := st.boundaryRegex.FindStringIndex(text)
	if loc != nil {
		return strings.TrimSpace(text[:loc[1]]), text[loc[1]:]
	}
	return "", text
}

func (st *sentenceTokenizer) Flush(ctx context.Context, contextId string) error {
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

func (st *sentenceTokenizer) flushUnsafe(ctx context.Context, contextId string) error {
	remaining := strings.TrimSpace(st.buffer.String())
	if remaining == "" {
		st.buffer.Reset()
		return nil
	}
	if remaining != "" {
		if err := st.callback(ctx, contextId, remaining); err != nil {
			return err
		}
	}
	st.buffer.Reset()
	return nil
}

func (st *sentenceTokenizer) GetCurrentBuffer() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.buffer.String()
}

func (st *sentenceTokenizer) GetCurrentContext() string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.currentContext
}

func (st *sentenceTokenizer) Reset() {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.buffer.Reset()
	st.currentContext = ""
}

func (st *sentenceTokenizer) Close() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.buffer.Reset()
	st.currentContext = ""
	return nil
}

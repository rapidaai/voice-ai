// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_normalizers

import (
	"strings"

	"github.com/rapidaai/pkg/commons"
)

type roleAbbreviationNormalizer struct {
	logger    commons.Logger
	abbrevMap map[string]string
}

func NewRoleAbbreviationNormalizer(logger commons.Logger) Normalizer {
	return &roleAbbreviationNormalizer{
		logger: logger,
		abbrevMap: map[string]string{
			"vp":       "vee pee",
			"v.p.":     "vee pee",
			"phd":      "pee aitch dee",
			"ph.d.":    "pee aitch dee",
			"r&d":      "are and dee",
			"hr":       "aitch are",
			"h.r.":     "aitch are",
			"ceo":      "see ee oh",
			"c.e.o.":   "see ee oh",
			"cfo":      "see ef oh",
			"c.f.o.":   "see ef oh",
			"coo":      "see oh oh",
			"c.o.o.":   "see oh oh",
			"cto":      "see tee oh",
			"c.t.o.":   "see tee oh",
			"cro":      "see are oh",
			"c.r.o.":   "see are oh",
			"cmo":      "see em oh",
			"c.m.o.":   "see em oh",
			"cio":      "see eye oh",
			"c.i.o.":   "see eye oh",
			"chro":     "see aitch are oh",
			"c.h.r.o.": "see aitch are oh",
		},
	}
}

func (ran *roleAbbreviationNormalizer) Normalize(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if expanded, ok := ran.abbrevMap[strings.ToLower(word)]; ok {
			words[i] = expanded
		}
	}
	return strings.Join(words, " ")
}

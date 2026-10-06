// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_output_processors

import (
	"strings"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
)

type generalAbbreviationProcessor struct {
	logger    commons.Logger
	abbrevMap map[string]string
}

func NewGeneralAbbreviationProcessor(logger commons.Logger) internal_type.TextProcessor {
	return &generalAbbreviationProcessor{
		logger: logger,
		abbrevMap: map[string]string{
			"aka":     "ay kay ay",
			"a.k.a.":  "ay kay ay",
			"pr":      "pee are",
			"p.r.":    "pee are",
			"rsvp":    "are es vee pee",
			"ps":      "pee ess",
			"p.s.":    "pee ess",
			"dr.":     "doctor",
			"mr.":     "mister",
			"mrs.":    "missus",
			"ms.":     "mizz",
			"jr.":     "junior",
			"sr.":     "senior",
			"rev.":    "reverend",
			"st.":     "saint",
			"ave.":    "avenue",
			"blvd.":   "boulevard",
			"ct.":     "court",
			"rd.":     "road",
			"sq.":     "square",
			"ln.":     "lane",
			"apt.":    "apartment",
			"dept.":   "department",
			"vs.":     "versus",
			"etc.":    "etcetera",
			"i.e.":    "that is",
			"e.g.":    "for example",
			"a.m.":    "ay em",
			"p.m.":    "pee em",
			"asap":    "ay sap",
			"approx.": "approximately",
			"est.":    "established",
			"min.":    "minimum",
			"max.":    "maximum",
			"misc.":   "miscellaneous",
			"vol.":    "volume",
			"yr.":     "year",
		},
	}
}

func (gan *generalAbbreviationProcessor) Process(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if expanded, ok := gan.abbrevMap[strings.ToLower(word)]; ok {
			words[i] = expanded
		}
	}
	return strings.Join(words, " ")
}

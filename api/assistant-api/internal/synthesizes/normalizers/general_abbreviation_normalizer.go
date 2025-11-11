package internal_normalizers

import (
	"strings"

	"github.com/rapidaai/pkg/commons"
)

type generalAbbreviationNormalizer struct {
	logger    commons.Logger
	abbrevMap map[string]string
}

func NewGeneralAbbreviationNormalizer(logger commons.Logger) Normalizer {
	return &generalAbbreviationNormalizer{
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

func (gan *generalAbbreviationNormalizer) Normalize(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if expanded, ok := gan.abbrevMap[strings.ToLower(word)]; ok {
			words[i] = expanded
		}
	}
	return strings.Join(words, " ")
}

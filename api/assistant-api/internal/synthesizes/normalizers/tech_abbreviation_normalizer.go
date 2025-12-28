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

type techAbbreviationNormalizer struct {
	logger    commons.Logger
	abbrevMap map[string]string
}

func NewTechAbbreviationNormalizer(logger commons.Logger) Normalizer {
	return &techAbbreviationNormalizer{
		logger: logger,
		abbrevMap: map[string]string{
			// only for rapida
			"rapida": "rahpidah",

			//
			"ai":       "eh eye",
			"a.i.":     "eh eye",
			"ml":       "em el",
			"m.l.":     "em el",
			"ui":       "you eye",
			"u.i.":     "you eye",
			"ux":       "you ex",
			"u.x.":     "you ex",
			"api":      "ay pee eye",
			"a.p.i.":   "ay pee eye",
			"html":     "aitch tee em el",
			"css":      "see es es",
			"js":       "jay ess",
			"db":       "dee bee",
			"os":       "oh ess",
			"iot":      "eye oh tee",
			"i.o.t.":   "eye oh tee",
			"saas":     "sass",
			"s.a.a.s.": "sass",
			"paas":     "pass",
			"p.a.a.s.": "pass",
			"iaas":     "eye ass",
			"i.a.a.s.": "eye ass",
			"erp":      "ee are pee",
			"e.r.p.":   "ee are pee",
			"crm":      "see are em",
			"c.r.m.":   "see are em",
			"rpa":      "are pee ay",
			"r.p.a.":   "are pee ay",
			"sql":      "ess queue el",
			"nosql":    "no ess queue el",
			"bi":       "bee eye",
			"b.i.":     "bee eye",
			"vr":       "vee are",
			"v.r.":     "vee are",
			"ar":       "ay are",
			"a.r.":     "ay are",
			"devops":   "dev ops",
			"ci/cd":    "see eye see dee",
			"cdn":      "see dee en",
			"c.d.n.":   "see dee en",
			"seo":      "ess ee oh",
			"s.e.o.":   "ess ee oh",
			"tcp/ip":   "tee see pee eye pee",
			"http":     "aitch tee tee pee",
			"https":    "aitch tee tee pee ess",
			"ftp":      "ef tee pee",
			"ssh":      "ess ess aitch",
			"vpn":      "vee pee en",
			"lan":      "lan",
			"wan":      "wan",
			"ram":      "ram",
			"rom":      "rom",
			"cpu":      "see pee you",
			"gpu":      "gee pee you",
			"ssd":      "ess ess dee",
			"hdd":      "aitch dee dee",
			"ide":      "eye dee ee",
			"gui":      "gooey",
			"cli":      "see el eye",
			"sdk":      "ess dee kay",
			"rest":     "rest",
			"soap":     "soap",
			"xml":      "ex em el",
			"json":     "jay son",
			"yaml":     "yam el",
			"smtp":     "ess em tee pee",
			"dns":      "dee en ess",
			"ip":       "eye pee",
			"voip":     "voy pee",
			"ai/ml":    "ay eye em el",
			"nlp":      "en el pee",
			"ocr":      "oh see are",
			"rfid":     "are ef eye dee",
			"cms":      "see em ess",
			"dms":      "dee em ess",
			"etl":      "ee tee el",
			"olap":     "oh lap",
			"oltp":     "oh el tee pee",
			"cicd":     "see eye see dee",
		},
	}
}

func (tan *techAbbreviationNormalizer) Normalize(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if expanded, ok := tan.abbrevMap[strings.ToLower(word)]; ok {
			words[i] = expanded
		}
	}
	return strings.Join(words, " ")
}

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package parsers

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"github.com/flosch/pongo2/v6"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
)

type pongo2TemplateParser struct {
	logger commons.Logger
}

type pongo2StringTemplateParser struct {
	pongo2TemplateParser
}

type pongo2MessageTemplateParser struct {
	pongo2TemplateParser
}

func NewPongo2StringTemplateParser(logger commons.Logger) StringTemplateParser {
	return &pongo2StringTemplateParser{
		pongo2TemplateParser: pongo2TemplateParser{logger: logger},
	}
}

func (stp *pongo2StringTemplateParser) Parse(template string, argument map[string]interface{}) string {
	tpl, err := pongo2.FromString(template)
	if err != nil {
		stp.logger.Errorf("error while parsing the template with pongo2: %v", err)
		return template
	}
	context := CanonicalizePromptArguments(utils.NormalizeInterface(argument))
	context = EncodeInterpolatedComposites(template, context)
	formattedTemplate, err := tpl.Execute(pongo2.Context(context))
	if err != nil {
		stp.logger.Errorf("error while executing the template with pongo2: %v", err)
		return template
	}
	return formattedTemplate
}

// EncodeInterpolatedComposites JSON-encodes composite values that the template
// interpolates directly, for example "{{ messages }}" over a slice of turns.
// pongo2 has no string form for a composite value and falls back to Go's debug
// formatting, so such a placeholder renders as "<[]interface {} Value>" and the
// data never reaches the prompt.
//
// Encoding is deliberately narrow, because composites are otherwise load-bearing:
//
//   - Templates containing tags are left untouched. "{% for m in messages %}"
//     iterates the composite itself and needs the original value.
//   - Only keys used as a standalone "{{ key }}" are encoded. Attribute access
//     such as "{{ message.language }}" still resolves against the live map.
func EncodeInterpolatedComposites(template string, in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 || strings.Contains(template, "{%") {
		return in
	}

	keys := bareInterpolationKeys(template)
	if len(keys) == 0 {
		return in
	}

	out := make(map[string]interface{}, len(in))
	for key, value := range in {
		out[key] = value

		if _, interpolated := keys[key]; !interpolated {
			continue
		}

		switch value.(type) {
		case []interface{}, map[string]interface{}:
			encoded, err := encodeCompositeValue(value)
			if err != nil {
				continue
			}
			// Marked safe so pongo2 autoescaping does not turn the quotes of
			// the encoded payload into HTML entities.
			out[key] = pongo2.AsSafeValue(encoded)
		}
	}

	return out
}

// encodeCompositeValue renders value as JSON without escaping "<", ">" and "&".
// The output is destined for a model prompt rather than a browser, and escaping
// those characters only makes a transcript harder for the model to read.
func encodeCompositeValue(value interface{}) (string, error) {
	var buffer bytes.Buffer

	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}

	// Encode always terminates the payload with a newline.
	return strings.TrimRight(buffer.String(), "\n"), nil
}

// bareInterpolationKeys collects identifiers used as a standalone "{{ key }}".
// Expressions carrying attribute access, filters or calls are skipped, since
// those resolve against the value's structure rather than its string form.
func bareInterpolationKeys(template string) map[string]struct{} {
	keys := make(map[string]struct{})

	for rest := template; ; {
		open := strings.Index(rest, "{{")
		if open < 0 {
			break
		}
		rest = rest[open+2:]

		end := strings.Index(rest, "}}")
		if end < 0 {
			break
		}

		expression := strings.TrimSpace(rest[:end])
		rest = rest[end+2:]

		if expression != "" && !strings.ContainsAny(expression, ".|()[] ") {
			keys[expression] = struct{}{}
		}
	}

	return keys
}

// canonicalizePromptArguments expands dotted top-level keys (for example
// "message.language") into nested maps so pongo2 receives valid identifiers.
// It processes plain keys first, then dotted keys in lexical order so
// conflicts resolve deterministically.
func CanonicalizePromptArguments(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return map[string]interface{}{}
	}

	out := make(map[string]interface{}, len(in))
	dottedKeys := make([]string, 0)

	for key, value := range in {
		if strings.Contains(key, ".") {
			dottedKeys = append(dottedKeys, key)
			continue
		}

		if nested, ok := value.(map[string]interface{}); ok {
			value = CanonicalizePromptArguments(nested)
		}
		out[key] = value
	}

	sort.Strings(dottedKeys)
	for _, key := range dottedKeys {
		value := in[key]
		if nested, ok := value.(map[string]interface{}); ok {
			value = CanonicalizePromptArguments(nested)
		}
		setNestedPromptArgument(out, strings.Split(key, "."), value)
	}

	return out
}

func setNestedPromptArgument(target map[string]interface{}, parts []string, value interface{}) {
	current := target
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			return
		}

		if i == len(parts)-1 {
			if _, exists := current[part]; exists {
				return
			}
			current[part] = value
			return
		}

		next, exists := current[part]
		if !exists {
			child := make(map[string]interface{})
			current[part] = child
			current = child
			continue
		}

		child, ok := next.(map[string]interface{})
		if !ok {
			child = make(map[string]interface{})
			current[part] = child
		}
		current = child
	}
}

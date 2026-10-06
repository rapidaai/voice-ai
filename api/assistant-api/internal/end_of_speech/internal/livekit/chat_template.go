// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

// chatMessage represents a single message in the conversation for templating.
type chatMessage struct {
	Role    string
	Content string
}

// formatChatTemplateFromHistory applies the selected LiveKit model's chat template.
// The last message is left open so the model can predict turn completion.
func formatChatTemplateFromHistory(history []chatMessage, currentText string, maxTurns int, modelType string) string {
	candidateMessages := make([]chatMessage, 0, len(history)+1)
	for _, historyMessage := range history {
		if historyMessage.Role != "user" && historyMessage.Role != "assistant" {
			continue
		}
		if historyMessage.Content == "" {
			continue
		}
		candidateMessages = append(candidateMessages, historyMessage)
	}
	if currentText != "" {
		candidateMessages = append(candidateMessages, chatMessage{Role: "user", Content: currentText})
	}

	if maxTurns > 0 && len(candidateMessages) > maxTurns {
		candidateMessages = candidateMessages[len(candidateMessages)-maxTurns:]
	}

	formattedMessages := make([]chatMessage, 0, len(candidateMessages))
	for _, candidateMessage := range candidateMessages {
		formattedContent := candidateMessage.Content
		if modelType == multilingualModelType {
			canonicalContent := norm.NFKC.String(cases.Lower(language.Und).String(candidateMessage.Content))
			var cleanContentBuilder strings.Builder
			cleanContentBuilder.Grow(len(canonicalContent))
			for _, contentRune := range canonicalContent {
				if unicode.IsPunct(contentRune) && contentRune != '\'' && contentRune != '-' {
					continue
				}
				// Python also treats these four control separators as whitespace.
				if unicode.IsSpace(contentRune) || contentRune >= '\x1c' && contentRune <= '\x1f' {
					cleanContentBuilder.WriteByte(' ')
					continue
				}
				cleanContentBuilder.WriteRune(contentRune)
			}
			formattedContent = strings.Join(strings.Fields(cleanContentBuilder.String()), " ")
		}
		if len(formattedMessages) > 0 && formattedMessages[len(formattedMessages)-1].Role == candidateMessage.Role {
			formattedMessages[len(formattedMessages)-1].Content += " " + formattedContent
			continue
		}
		formattedMessages = append(formattedMessages, chatMessage{Role: candidateMessage.Role, Content: formattedContent})
	}

	if len(formattedMessages) == 0 {
		return ""
	}

	var templateBuilder strings.Builder
	templateBuilder.Grow(256)

	for messageIndex, formattedMessage := range formattedMessages {
		templateBuilder.WriteString("<|im_start|>")
		if modelType == multilingualModelType {
			templateBuilder.WriteString(formattedMessage.Role)
			templateBuilder.WriteByte('\n')
		} else {
			templateBuilder.WriteString("<|")
			templateBuilder.WriteString(formattedMessage.Role)
			templateBuilder.WriteString("|>")
		}
		templateBuilder.WriteString(formattedMessage.Content)
		if messageIndex != len(formattedMessages)-1 {
			templateBuilder.WriteString("<|im_end|>")
			if modelType == multilingualModelType {
				templateBuilder.WriteByte('\n')
			}
		}
	}

	return templateBuilder.String()
}

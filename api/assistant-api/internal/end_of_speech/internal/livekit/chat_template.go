// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// chatMessage represents a single message in the conversation for templating.
type chatMessage struct {
	Role    string
	Content string
}

// formatChatTemplateFromHistory formats history for the SmolLM2 chat template.
// The last user message is left open so the model can predict turn completion.
func formatChatTemplateFromHistory(history []chatMessage, currentText string, maxTurns int) string {
	candidateMessages := make([]chatMessage, 0, len(history)+1)
	for _, historyMessage := range history {
		if historyMessage.Role == "" || strings.TrimSpace(historyMessage.Content) == "" {
			continue
		}
		candidateMessages = append(candidateMessages, historyMessage)
	}
	if strings.TrimSpace(currentText) != "" {
		candidateMessages = append(candidateMessages, chatMessage{Role: "user", Content: currentText})
	}

	if maxTurns > 0 && len(candidateMessages) > maxTurns {
		candidateMessages = candidateMessages[len(candidateMessages)-maxTurns:]
	}

	formattedMessages := make([]chatMessage, 0, len(candidateMessages))
	for _, candidateMessage := range candidateMessages {
		canonicalContent := strings.ToLower(norm.NFKC.String(candidateMessage.Content))
		var cleanContentBuilder strings.Builder
		cleanContentBuilder.Grow(len(canonicalContent))
		for _, contentRune := range canonicalContent {
			if unicode.IsPunct(contentRune) && contentRune != '\'' && contentRune != '-' {
				continue
			}
			if unicode.IsSpace(contentRune) {
				cleanContentBuilder.WriteByte(' ')
				continue
			}
			cleanContentBuilder.WriteRune(contentRune)
		}
		cleanContent := strings.Join(strings.Fields(cleanContentBuilder.String()), " ")
		if cleanContent == "" {
			continue
		}
		if len(formattedMessages) > 0 && formattedMessages[len(formattedMessages)-1].Role == candidateMessage.Role {
			formattedMessages[len(formattedMessages)-1].Content += " " + cleanContent
			continue
		}
		formattedMessages = append(formattedMessages, chatMessage{Role: candidateMessage.Role, Content: cleanContent})
	}

	if len(formattedMessages) == 0 {
		return ""
	}

	var templateBuilder strings.Builder
	templateBuilder.Grow(256)

	for messageIndex, formattedMessage := range formattedMessages {
		templateBuilder.WriteString("<|im_start|>")
		templateBuilder.WriteString(formattedMessage.Role)
		templateBuilder.WriteByte('\n')
		templateBuilder.WriteString(formattedMessage.Content)
		if messageIndex != len(formattedMessages)-1 {
			templateBuilder.WriteString("<|im_end|>")
			templateBuilder.WriteByte('\n')
		}
	}

	return templateBuilder.String()
}

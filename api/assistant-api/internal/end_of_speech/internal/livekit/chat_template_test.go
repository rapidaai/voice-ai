// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_livekit

import (
	"slices"
	"testing"
)

// Fixtures follow LiveKit agents ab6e8a1ca, turn_detector/base.py and english.py.
func TestFormatChatTemplateFromHistory_ReferenceParity(t *testing.T) {
	tests := []struct {
		name        string
		history     []chatMessage
		currentText string
		maxTurns    int
		modelType   string
		want        string
	}{
		{
			name:      "empty context",
			maxTurns:  6,
			modelType: defaultModelType,
		},
		{
			name: "empty and unsupported history omitted",
			history: []chatMessage{
				{Role: "system", Content: "instructions"},
				{Role: "tool", Content: "result"},
				{Role: "", Content: "missing role"},
				{Role: "user", Content: ""},
				{Role: "assistant", Content: ""},
			},
			maxTurns:  6,
			modelType: defaultModelType,
		},
		{
			name:        "english preserves all content bytes",
			currentText: " \tHello,\n  THERE!\r\n\u00a0\uff21\u0130\u2019\x1c ",
			maxTurns:    6,
			modelType:   defaultModelType,
			want:        "<|im_start|><|user|> \tHello,\n  THERE!\r\n\u00a0\uff21\u0130\u2019\x1c ",
		},
		{
			name:        "english whitespace current user",
			history:     []chatMessage{{Role: "assistant", Content: "OK."}},
			currentText: " \t\n ",
			maxTurns:    6,
			modelType:   defaultModelType,
			want:        "<|im_start|><|assistant|>OK.<|im_end|><|im_start|><|user|> \t\n ",
		},
		{
			name: "english merge inserts exactly one space",
			history: []chatMessage{
				{Role: "user", Content: "Hello! \n"},
				{Role: "user", Content: "\tAgain? "},
			},
			currentText: " More.",
			maxTurns:    6,
			modelType:   defaultModelType,
			want:        "<|im_start|><|user|>Hello! \n \tAgain?   More.",
		},
		{
			name: "english whitespace history separates roles",
			history: []chatMessage{
				{Role: "user", Content: "First"},
				{Role: "assistant", Content: " \t"},
			},
			currentText: "Next",
			maxTurns:    6,
			modelType:   defaultModelType,
			want:        "<|im_start|><|user|>First<|im_end|><|im_start|><|assistant|> \t<|im_end|><|im_start|><|user|>Next",
		},
		{
			name:        "multilingual lowercase precedes NFKC",
			currentText: "\U0001d400 \u2102 \uff21",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\nA C a",
		},
		{
			name:        "multilingual expanded and contextual lowercase",
			currentText: "\u0130 \u039f\u03a3 \u039f\u03a3\u0391",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\ni\u0307 \u03bf\u03c2 \u03bf\u03c3\u03b1",
		},
		{
			name:        "multilingual punctuation and whitespace",
			currentText: " \t\uff28\uff25\uff2c\uff2c\uff2f\uff01\n I'M still-talking... \u2018Yes\u2019\u2014OK +$ \u00a0",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\nhello i'm still-talking yesok +$",
		},
		{
			name:        "multilingual Python whitespace separators",
			currentText: "\x1cA\x1d\x1eB\x1fC\u0085D\u2028E\u2029\u3000",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\na b c d e",
		},
		{
			name:        "punctuation current user retains role after assistant",
			history:     []chatMessage{{Role: "assistant", Content: "Ready?"}},
			currentText: "...?!",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>assistant\nready<|im_end|>\n<|im_start|>user\n",
		},
		{
			name:        "whitespace current user retains role after assistant",
			history:     []chatMessage{{Role: "assistant", Content: "Ready?"}},
			currentText: " \t\n",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>assistant\nready<|im_end|>\n<|im_start|>user\n",
		},
		{
			name:        "cleaned empty current user alone",
			currentText: "...",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\n",
		},
		{
			name:        "cleaned empty current user merges trailing space",
			history:     []chatMessage{{Role: "user", Content: "Wait!"}},
			currentText: "...",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\nwait ",
		},
		{
			name: "cleaned empty history preserves role boundaries",
			history: []chatMessage{
				{Role: "user", Content: "Hi"},
				{Role: "assistant", Content: "?!"},
			},
			currentText: "Next",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\nhi<|im_end|>\n<|im_start|>assistant\n<|im_end|>\n<|im_start|>user\nnext",
		},
		{
			name: "cleaned empty adjacent history keeps merge spaces",
			history: []chatMessage{
				{Role: "user", Content: "..."},
				{Role: "user", Content: "?!"},
			},
			currentText: "Next",
			maxTurns:    6,
			modelType:   multilingualModelType,
			want:        "<|im_start|>user\n  next",
		},
		{
			name: "filter roles and empty content before selection",
			history: []chatMessage{
				{Role: "user", Content: "Old"},
				{Role: "assistant", Content: "Recent"},
				{Role: "system", Content: "Ignore"},
				{Role: "user", Content: ""},
				{Role: "tool", Content: "Ignore"},
				{Role: "Assistant", Content: "Ignore"},
			},
			currentText: "Now",
			maxTurns:    2,
			modelType:   defaultModelType,
			want:        "<|im_start|><|assistant|>Recent<|im_end|><|im_start|><|user|>Now",
		},
		{
			name: "select before merging adjacent roles",
			history: []chatMessage{
				{Role: "assistant", Content: "Old reply"},
				{Role: "user", Content: "Old user"},
				{Role: "user", Content: "Recent"},
			},
			currentText: "Now",
			maxTurns:    2,
			modelType:   defaultModelType,
			want:        "<|im_start|><|user|>Recent Now",
		},
		{
			name: "select before cleaning empty content",
			history: []chatMessage{
				{Role: "user", Content: "Old"},
				{Role: "assistant", Content: "..."},
			},
			currentText: "Now",
			maxTurns:    2,
			modelType:   multilingualModelType,
			want:        "<|im_start|>assistant\n<|im_end|>\n<|im_start|>user\nnow",
		},
		{
			name: "whitespace counts toward selected history",
			history: []chatMessage{
				{Role: "assistant", Content: "Old"},
				{Role: "user", Content: " \t"},
			},
			currentText: "Now",
			maxTurns:    2,
			modelType:   defaultModelType,
			want:        "<|im_start|><|user|> \t Now",
		},
		{
			name: "filtering permits adjacent assistant merge",
			history: []chatMessage{
				{Role: "assistant", Content: "One"},
				{Role: "system", Content: "Ignore"},
				{Role: "user", Content: ""},
				{Role: "assistant", Content: "Two"},
			},
			currentText: "Three",
			maxTurns:    6,
			modelType:   defaultModelType,
			want:        "<|im_start|><|assistant|>One Two<|im_end|><|im_start|><|user|>Three",
		},
		{
			name: "empty current content leaves last assistant open",
			history: []chatMessage{
				{Role: "user", Content: "Old"},
				{Role: "assistant", Content: "Reply"},
			},
			maxTurns:  1,
			modelType: defaultModelType,
			want:      "<|im_start|><|assistant|>Reply",
		},
		{
			name:        "single turn selection includes current user",
			history:     []chatMessage{{Role: "assistant", Content: "Old"}},
			currentText: "Now",
			maxTurns:    1,
			modelType:   defaultModelType,
			want:        "<|im_start|><|user|>Now",
		},
		{
			name:        "zero history limit retains existing unbounded behavior",
			history:     []chatMessage{{Role: "assistant", Content: "Reply"}},
			currentText: "Now",
			modelType:   defaultModelType,
			want:        "<|im_start|><|assistant|>Reply<|im_end|><|im_start|><|user|>Now",
		},
		{
			name:        "negative history limit retains existing unbounded behavior",
			history:     []chatMessage{{Role: "assistant", Content: "Reply"}},
			currentText: "Now",
			maxTurns:    -1,
			modelType:   defaultModelType,
			want:        "<|im_start|><|assistant|>Reply<|im_end|><|im_start|><|user|>Now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			historyBefore := slices.Clone(tt.history)
			got := formatChatTemplateFromHistory(tt.history, tt.currentText, tt.maxTurns, tt.modelType)
			if got != tt.want {
				t.Errorf("formatted history = %q, want %q", got, tt.want)
			}
			if !slices.Equal(tt.history, historyBefore) {
				t.Errorf("history mutated: got %#v, want %#v", tt.history, historyBefore)
			}
		})
	}
}

/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */
package types

import (
	"strings"
)

// Emotion represents a standardized user emotion
type Emotion struct {
	Name  string // e.g., "anger", "positivity", "sadness"
	Level string // e.g., "low", "high", "moderate"
}

// GetEmotionFromString maps raw user emotion strings to structured Emotion
func GetEmotionByName(raw string) Emotion {
	rawLower := strings.ToLower(strings.TrimSpace(raw))
	switch rawLower {
	case "angry", "annoyed", "irritated", "mad":
		return Emotion{"anger", "high"}
	case "frustrated":
		return Emotion{"anger", "low"}
	case "sad", "worried", "disappointed", "upset":
		return Emotion{"sadness", "high"}
	case "confused", "skeptical":
		return Emotion{"curiosity", "low"}
	case "curious", "interested":
		return Emotion{"curiosity", ""}
	case "happy", "excited", "grateful", "joyful":
		return Emotion{"positivity", "high"}
	case "content", "relieved", "calm":
		return Emotion{"positivity", "low"}
	default:
		return Emotion{"neutral", ""}
	}
}

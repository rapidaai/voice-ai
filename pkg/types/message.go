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
	"time"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

func OnlyStringContent(Contents []*Content) string {
	var builder strings.Builder
	if len(Contents) == 0 {
		return ""
	}
	for _, c := range Contents {
		if commons.ResponseContentType(c.GetContentType()) == commons.TEXT_CONTENT {
			if commons.ResponseContentFormat(c.GetContentFormat()) == commons.TEXT_CONTENT_FORMAT_RAW {
				builder.Write(c.Content)
			}
		}
	}
	return builder.String()
}

func ContentString(c *lexatic_backend.Content) string {
	var builder strings.Builder
	if commons.ResponseContentType(c.GetContentType()) == commons.TEXT_CONTENT {
		if commons.ResponseContentFormat(c.GetContentFormat()) == commons.TEXT_CONTENT_FORMAT_RAW {
			builder.Write(c.Content)
		}
	}
	return builder.String()
}

func OnlyStringProtoContent(Contents []*lexatic_backend.Content) string {
	var builder strings.Builder
	if len(Contents) == 0 {
		return ""
	}
	for _, c := range Contents {
		if commons.ResponseContentType(c.GetContentType()) == commons.TEXT_CONTENT {
			if commons.ResponseContentFormat(c.GetContentFormat()) == commons.TEXT_CONTENT_FORMAT_RAW {
				builder.Write(c.Content)
			}
		}
	}
	return builder.String()
}

func ContainsAudioContent(Contents []*lexatic_backend.Content) bool {
	for _, c := range Contents {
		if commons.ResponseContentType(c.GetContentType()) == commons.AUDIO_CONTENT {
			return true
		}
	}
	return false
}

func ToMessage(msg *lexatic_backend.Message) *Message {
	out := &Message{}
	err := utils.Cast(msg, out)
	if err != nil {
		return nil
	}
	out.Time = time.Now()
	return out
}

func ToMessages(msgs []*lexatic_backend.Message) []*Message {
	out := make([]*Message, 0, len(msgs))
	for _, msg := range msgs {
		if convertedMsg := ToMessage(msg); convertedMsg != nil {
			out = append(out, convertedMsg)
		}
	}
	return out
}

func ToSimpleMessage(msgs []*Message) []map[string]string {
	out := make([]map[string]string, 0)
	for _, msg := range msgs {
		stringContent := OnlyStringContent(msg.GetContents())
		if strings.TrimSpace(stringContent) != "" {
			out = append(out, map[string]string{
				"role":    msg.GetRole(),
				"message": OnlyStringContent(msg.GetContents()),
				"time":    msg.GetTime(),
			})
		}
	}
	return out
}

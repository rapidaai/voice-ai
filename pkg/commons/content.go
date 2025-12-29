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

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package commons

const (
	TEXT_CONTENT        ResponseContentType = "text"
	AUDIO_CONTENT       ResponseContentType = "audio"
	IMAGE_CONTENT       ResponseContentType = "image"
	MULTI_MEDIA_CONTENT ResponseContentType = "multi"
)

const (
	// raw string in byte format
	TEXT_CONTENT_FORMAT_RAW ResponseContentFormat = "raw"

	//
	AUDIO_CONTENT_FORMAT_RAW ResponseContentFormat = "raw"
	AUDIO_CONTENT_FORMAT_URL ResponseContentFormat = "url"

	//
	IMAGE_CONTENT_FORMAT_RAW ResponseContentFormat = "raw"
	IMAGE_CONTENT_FORMAT_URL ResponseContentFormat = "url"

	MULTI_MEDIA_CONTENT_FORMAT_RAW ResponseContentFormat = "raw"
	MULTI_MEDIA_CONTENT_FORMAT_URL ResponseContentFormat = "url"
)

type MessageFinishReason string

const (
	MessageFinishReasonContentFiltered   MessageFinishReason = "content_filter"
	MessageFinishReasonFunctionCall      MessageFinishReason = "function_call"
	MessageFinishReasonStopped           MessageFinishReason = "stop"
	MessageFinishReasonTokenLimitReached MessageFinishReason = "length"
	MessageFinishReasonToolCalls         MessageFinishReason = "tool_calls"
)

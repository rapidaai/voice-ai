package utils

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

type RapidaEvent string

const (
	// signals to stop speaking
	// what happend is when user unintentionally interrupt the the voice completly stop
	// pause will make sure that it's not unintentionally interrupted
	TalkPause        RapidaEvent = "talk.onPause"
	TalkInterruption RapidaEvent = "talk.onInterrupt"

	//
	TalkTranscript RapidaEvent = "talk.onTranscript"
	// start and complete
	TalkStart    RapidaEvent = "talk.onStart"
	TalkComplete RapidaEvent = "talk.onComplete"

	TalkGeneration         RapidaEvent = "talk.onGeneration"
	TalkCompleteGeneration RapidaEvent = "talk.onCompleteGeneration"

	TalkStartConversation    RapidaEvent = "talk.onStartConversation"
	TalkCompleteConversation RapidaEvent = "talk.onCompleteConversation"
)

// Get returns the string value of the RapidaStage
func (r RapidaEvent) Get() string {
	return string(r)
}

type AssistantServerEvent string

const (
	AssistantInitiated AssistantWebhookEvent = "conversation.initiated"
	// Triggered when a new conversation is started.
)

func (r AssistantServerEvent) Get() string {
	return string(r)
}

type AssistantWebhookEvent string

const (
	MessageReceived AssistantWebhookEvent = "message.received"
	// Triggered when a message is received.

	MessageSent AssistantWebhookEvent = "message.sent"
	// Triggered when a mes

	ConversationBegin     AssistantWebhookEvent = "conversation.begin"
	ConversationResume    AssistantWebhookEvent = "conversation.resume"
	ConversationCompleted AssistantWebhookEvent = "conversation.completed"
	// Triggered when a conversation ends successfully.

	ConversationFailed AssistantWebhookEvent = "conversation.failed"
	// Triggered when a conversation encounters an error.

)

func (r AssistantWebhookEvent) Get() string {
	return string(r)
}

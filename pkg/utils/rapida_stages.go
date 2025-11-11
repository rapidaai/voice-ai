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
type RapidaStage string

const (
	AssistantConnectStage RapidaStage = "talk.assistant.connect"

	AssistantCreateConversationStage RapidaStage = "talk.assistant.connect.create-conversation"
	AssistantResumeConverstaionStage RapidaStage = "talk.assistant.connect.resume-conversation"

	AssistantListenConnectStage       RapidaStage = "talk.assistant.listen.connect"
	AssistantSpeakConnectStage        RapidaStage = "talk.assistant.speak.connect"
	AssistantListeningStage           RapidaStage = "talk.assistant.listen.listening"
	AssistantUtteranceStage           RapidaStage = "talk.assistant.utterance"
	AssistantInterruptStage           RapidaStage = "talk.assistant.interrupt"
	AssistantAgentConnectStage        RapidaStage = "talk.assistant.agent.connect"
	AssistantToolConnectStage         RapidaStage = "talk.assistant.tool.connect"
	AssistantToolExecuteStage         RapidaStage = "talk.assistant.tool.execute"
	AssistantAgentTextGenerationStage RapidaStage = "talk.assistant.agent.text-generation"
	AssistantTranscribeStage          RapidaStage = "talk.assistant.speak.transcribe"
	AssistantSpeakingStage            RapidaStage = "talk.assistant.speak.speaking"
	AssistantNotifyStage              RapidaStage = "talk.assistant.notify"
	AssistantDisconnectStage          RapidaStage = "talk.assistant.disconnect"
)

// Get returns the string value of the RapidaStage
func (r RapidaStage) Get() string {
	return string(r)
}

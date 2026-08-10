// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package options

const (
	ListenOptionLanguage        = "listen.language"
	ListenOptionModel           = "listen.model"
	ListenOptionThreshold       = "listen.threshold"
	ListenOptionAudioEncoding   = "listen.audio.encoding"
	ListenOptionAudioSampleRate = "listen.audio.sample_rate"
	ListenOptionQueryParams     = "listen.query_params"
	ListenOptionRequestRules    = "listen.request_rules"
	ListenOptionResponseRules   = "listen.response_rules"
)

const (
	ListenOptionRegion         = "listen.region"
	ListenOptionSmartFormat    = "listen.smart_format"
	ListenOptionFillerWords    = "listen.filler_words"
	ListenOptionVADEvents      = "listen.vad_events"
	ListenOptionEndpointing    = "listen.endpointing"
	ListenOptionPunctuate      = "listen.punctuate"
	ListenOptionMultichannel   = "listen.multichannel"
	ListenOptionKeyword        = "listen.keyword"
	ListenOptionOperatingPoint = "listen.operating_point"
)

const (
	ListenOptionWordTimestamps     = "listen.word_timestamps"
	ListenOptionSentenceTimestamps = "listen.sentence_timestamps"
	ListenOptionDiarize            = "listen.diarize"
	ListenOptionRedactPII          = "listen.redact_pii"
	ListenOptionRedactPCI          = "listen.redact_pci"
	ListenOptionNumerals           = "listen.numerals"
)

const (
	MicrophoneOptionBargeInTrigger          = "microphone.barge_in_trigger"
	MicrophoneLegacyVADOptionBargeInTrigger = "microphone.vad.barge_in_trigger"
)

const (
	BargeInTriggerVAD  = "vad"
	BargeInTriggerWord = "word"
)

const (
	SpeakOptionVoiceID         = "speak.voice.id"
	SpeakOptionLanguage        = "speak.language"
	SpeakOptionModel           = "speak.model"
	SpeakOptionAudioEncoding   = "speak.audio.encoding"
	SpeakOptionAudioSampleRate = "speak.audio.sample_rate"
	SpeakOptionQueryParams     = "speak.query_params"
	SpeakOptionRequestRules    = "speak.request_rules"
	SpeakOptionResponseRules   = "speak.response_rules"
)

const (
	SpeakOptionExperimentalControlsSpeed   = "speak.__experimental_controls.speed"
	SpeakOptionExperimentalControlsEmotion = "speak.__experimental_controls.emotion"
	SpeakOptionSpeedAlpha                  = "speak.speed_alpha"
	SpeakOptionSpeed                       = "speak.speed"
)

const (
	SpeakerOptionLanguage                  = "speaker.language"
	SpeakerOptionVoiceName                 = "speaker.voice.name"
	SpeakerOptionConjunctionBoundaries     = "speaker.conjunction.boundaries"
	SpeakerOptionConjunctionBreak          = "speaker.conjunction.break"
	SpeakerOptionPronunciationDictionaries = "speaker.pronunciation.dictionaries"
	SpeakerOptionAmbient                   = "speaker.ambient"
	SpeakerOptionAmbientVolume             = "speaker.ambient_volume"
)

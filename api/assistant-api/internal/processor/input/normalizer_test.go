// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_input_processors

import (
	"context"
	"testing"

	internal_type "github.com/rapidaai/api/assistant-api/internal/type"
	"github.com/rapidaai/pkg/commons"
	rapida_types "github.com/rapidaai/pkg/types"
)

type parserStub struct {
	calls int
	out   rapida_types.Language
	conf  float64
}

func (p *parserStub) Parse(_ string) (rapida_types.Language, float64) {
	p.calls++
	return p.out, p.conf
}

func newTestProcessor(t *testing.T, onPacket func(context.Context, ...internal_type.Packet) error) *inputProcessor {
	t.Helper()
	logger, _ := commons.NewApplicationLogger()
	n := NewInputProcessor(logger).(*inputProcessor)
	n.onPacket = onPacket
	return n
}

func mustLanguage(t *testing.T, code string) rapida_types.Language {
	t.Helper()
	lang := rapida_types.LookupLanguage(code)
	if lang == rapida_types.UNKNOWN_LANGUAGE && code != "unknown" {
		t.Fatalf("language %q not found", code)
	}
	return lang
}

func TestInputProcessor_Process_EndOfSpeechBuildsUserInputPacket(t *testing.T) {
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	packets := []internal_type.Packet{
		internal_type.EndOfSpeechPacket{
			ContextID: "ctx-1",
			Speech:    "hello there",
			Speechs: []internal_type.SpeechToTextPacket{
				{ContextID: "ctx-1", Script: "hello", Language: "en"},
				{ContextID: "ctx-1", Script: "there", Language: "en-US"},
				{ContextID: "ctx-1", Script: "bonjour", Language: "fr"},
			},
		},
	}
	if err := n.Process(context.Background(), packets...); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emitted) != 1 {
		t.Fatalf("expected one emitted packet, got %d", len(emitted))
	}
	out, ok := emitted[0].(internal_type.UserInputPacket)
	if !ok {
		t.Fatalf("expected UserInputPacket, got %T", emitted[0])
	}
	if out.ContextID != "ctx-1" {
		t.Fatalf("expected context ctx-1, got %q", out.ContextID)
	}
	if out.Text != "hello there" {
		t.Fatalf("expected speech text preserved, got %q", out.Text)
	}
	if out.Language.ISO639_1 != "en" {
		t.Fatalf("expected consensus language en, got %q", out.Language.ISO639_1)
	}
}

func TestInputProcessor_Process_UserTextUsesParserWhenNoChunkLanguage(t *testing.T) {
	parser := &parserStub{out: mustLanguage(t, "es"), conf: 0.99}
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	n.parser = parser

	if err := n.Process(context.Background(), internal_type.UserTextReceivedPacket{ContextID: "ctx-2", Text: "hola como estas"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parser.calls != 1 {
		t.Fatalf("expected parser called once, got %d", parser.calls)
	}
	out := emitted[0].(internal_type.UserInputPacket)
	if out.Language.ISO639_1 != "es" {
		t.Fatalf("expected parser language es, got %q", out.Language.ISO639_1)
	}
}

func TestInputProcessor_Process_UserTextUsesProvidedLanguage(t *testing.T) {
	parser := &parserStub{out: mustLanguage(t, "en"), conf: 0.99}
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	n.parser = parser

	if err := n.Process(context.Background(), internal_type.UserTextReceivedPacket{ContextID: "ctx-2", Text: "hola como estas", Language: "fr"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parser.calls != 0 {
		t.Fatalf("expected parser not called when language present, got %d", parser.calls)
	}
	out := emitted[0].(internal_type.UserInputPacket)
	if out.Language.ISO639_1 != "fr" {
		t.Fatalf("expected canonical language fr, got %q", out.Language.ISO639_1)
	}
}

func TestInputProcessor_Process_UnknownChunkLanguageFallsBackToUnknownOnTie(t *testing.T) {
	parser := &parserStub{out: mustLanguage(t, "es"), conf: 0.99}
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	n.parser = parser

	err := n.Process(context.Background(), internal_type.EndOfSpeechPacket{
		ContextID: "ctx-3",
		Speech:    "bonjour",
		Speechs: []internal_type.SpeechToTextPacket{
			{ContextID: "ctx-3", Script: "??", Language: "xx"},
			{ContextID: "ctx-3", Script: "bonjour", Language: "fr-FR"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parser.calls != 0 {
		t.Fatalf("expected parser not called when valid chunk language exists, got %d", parser.calls)
	}
	out := emitted[0].(internal_type.UserInputPacket)
	if out.Language.ISO639_1 != "unknown" {
		t.Fatalf("expected language unknown, got %q", out.Language.ISO639_1)
	}
}

func TestInputProcessor_Process_ParserNoMatchUsesUnknownLanguage(t *testing.T) {
	parser := &parserStub{out: rapida_types.UNKNOWN_LANGUAGE}
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	n.parser = parser

	if err := n.Process(context.Background(), internal_type.UserTextReceivedPacket{ContextID: "ctx-4", Text: "??"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parser.calls != 1 {
		t.Fatalf("expected parser called once, got %d", parser.calls)
	}
	out := emitted[0].(internal_type.UserInputPacket)
	if out.Language.ISO639_1 != "unknown" {
		t.Fatalf("expected unknown language, got %q", out.Language.ISO639_1)
	}
}

func TestInputProcessor_Run_InputToOutputEmitsPacket(t *testing.T) {
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	n.Run(context.Background(), InputPipeline{ContextID: "ctx", Speech: "hello"})
	if len(emitted) != 1 {
		t.Fatalf("expected one emission, got %d", len(emitted))
	}
}

func TestInputProcessor_Run_DetectLanguageToOutputEmitsPacket(t *testing.T) {
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	n.Run(context.Background(), DetectLanguagePipeline{ContextID: "ctx", Speech: "hello"})
	if len(emitted) != 1 {
		t.Fatalf("expected one emission, got %d", len(emitted))
	}
}

func TestInputProcessor_Run_OutputEmitsPacket(t *testing.T) {
	emitted := make([]internal_type.Packet, 0)
	n := newTestProcessor(t, func(_ context.Context, pkts ...internal_type.Packet) error {
		emitted = append(emitted, pkts...)
		return nil
	})
	n.Run(context.Background(), OutputPipeline{ContextID: "ctx", Speech: "hello", Language: mustLanguage(t, "en")})
	if len(emitted) != 1 {
		t.Fatalf("expected one emission, got %d", len(emitted))
	}
}

func TestInputProcessor_Run_NilOnPacketNoOp(t *testing.T) {
	n := newTestProcessor(t, nil)
	n.onPacket = nil
	n.Run(context.Background(), OutputPipeline{ContextID: "ctx", Speech: "hello", Language: mustLanguage(t, "en")})
}

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"context"
	"sync"
	"time"

	internal_adapter_transformer_factory "github.com/rapidaai/api/assistant-api/internal/factory/transformer"
	internal_synthesizers "github.com/rapidaai/api/assistant-api/internal/synthesizes"
	internal_adapter_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	internal_tokenizer "github.com/rapidaai/api/assistant-api/internal/tokenizer"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

func (spk *GenericRequestor) FinishSpeaking(
	contextId string,
) error {
	_, err := spk.
		GetTextToSpeechTransformer()
	if err != nil {
		spk.logger.Warnf("no output transformer, skipping finish speak.")
		return err
	}

	ctx, span, _ := spk.Tracer().StartSpan(spk.Context(), utils.AssistantSpeakingStage)
	defer span.EndSpan(ctx, utils.AssistantSpeakingStage)

	span.AddAttributes(ctx,
		internal_adapter_telemetry.KV{
			K: "messageId", V: internal_adapter_telemetry.StringValue(contextId),
		},
		internal_adapter_telemetry.KV{
			K: "activity", V: internal_adapter_telemetry.StringValue("finish_speaking"),
		},
	)
	spk.
		tokenizer.
		Tokenize(
			ctx,
			contextId,
			"",
			true,
		)
	// keep it sync or blocking
	if spk.textToSpeechTransformer != nil {
		spk.textToSpeechTransformer.Transform(ctx,
			"",
			&internal_transformer.TextToSpeechOption{
				ContextId:  contextId,
				IsComplete: true,
			})
	}
	return nil

}

func (spk *GenericRequestor) Speak(contextId string, msg string) error {
	if _, err := spk.GetTextToSpeechTransformer(); err != nil {
		spk.logger.Warnf("no output transformer, skipping speak")
		return err
	}
	ctx, span, _ := spk.Tracer().StartSpan(spk.Context(), utils.AssistantTranscribeStage)
	defer span.EndSpan(ctx, utils.AssistantTranscribeStage)
	span.AddAttributes(ctx,
		internal_adapter_telemetry.KV{
			K: "messageId", V: internal_adapter_telemetry.StringValue(contextId),
		},
		internal_adapter_telemetry.KV{
			K: "chunk", V: internal_adapter_telemetry.StringValue(msg),
		},
	)
	return spk.tokenizer.Tokenize(ctx, contextId, msg, false)

}

// Init initializes the audio talking system for a given assistant persona.
// It sets up both audio input and output transformer.
// This function is typically called at the beginning of a communication session.
func (spk *GenericRequestor) ConnectSpeaker(ctx context.Context, audioOutConfig *protos.AudioConfig) error {
	context, span, _ := spk.Tracer().StartSpan(ctx, utils.AssistantSpeakConnectStage)
	defer span.EndSpan(context, utils.AssistantSpeakConnectStage)

	start := time.Now()
	outputTransformer, err := spk.GetTextToSpeechTransformer()
	if err != nil {
		spk.logger.Errorf("no output transformer, so skipping it or error occured %v", err)
		return err
	}
	//

	speakerOpts := utils.MergeMaps(outputTransformer.GetOptions(), spk.GetOptions())
	span.AddAttributes(context,
		internal_adapter_telemetry.KV{
			K: "options", V: internal_adapter_telemetry.JSONValue(speakerOpts),
		},
		internal_adapter_telemetry.KV{
			K: "provider", V: internal_adapter_telemetry.StringValue(outputTransformer.AudioProvider),
		},
	)
	//
	var wg sync.WaitGroup
	wg.Add(1)
	utils.Go(context, func() {
		defer wg.Done()
		if tokenizer, err := internal_tokenizer.NewSentenceTokenizer(spk.logger, spk.OnCompleteSentence, speakerOpts); err == nil {
			spk.tokenizer = tokenizer
		}
		if normalizer, err := internal_synthesizers.NewSentenceNormalizeSynthesizer(spk.logger, internal_synthesizers.SynthesizerOptions{SpeakerOptions: speakerOpts}); err == nil {
			spk.synthesizers = append(spk.synthesizers, normalizer)
		}
		if formatter, err := internal_synthesizers.NewSentenceFormattingSynthesizer(spk.logger, internal_synthesizers.SynthesizerOptions{SpeakerOptions: speakerOpts}); err == nil {
			spk.synthesizers = append(spk.synthesizers, formatter)
		}
		spk.logger.Benchmark("speak.GetAudioOutputTransformer.synthesizers", time.Since(start))
	})

	wg.Add(1)
	utils.Go(context, func() {
		defer wg.Done()
		opts := &internal_transformer.TextToSpeechInitializeOptions{
			AudioConfig: audioOutConfig,
			OnSpeech: func(contextId string, v []byte) error {
				return spk.OutputAudio(contextId, v, false)
			},
			OnComplete: func(contextId string) error {
				return spk.OutputAudio(contextId, nil, true)
			},
			ModelOptions: speakerOpts,
		}

		credentialId, err := opts.ModelOptions.GetUint64("rapida.credential_id")
		if err != nil {
			spk.logger.Errorf("unable to find credential from options %+v", err)
			return
		}
		credential, err := spk.
			VaultCaller().
			GetCredential(context, spk.Auth(), credentialId)
		if err != nil {
			spk.logger.Errorf("Api call to find credential failed %+v", err)
			return
		}

		atransformer, err := internal_adapter_transformer_factory.GetTextToSpeechTransformer(internal_adapter_transformer_factory.AudioTransformer(outputTransformer.GetName()), context, spk.logger, credential, opts)
		if err != nil {
			spk.logger.Errorf("unable to create input audio transformer with error %v", err)
			return
		}
		spk.logger.Benchmark("speak.transformer.GetOutputAudioTransformer", time.Since(start))
		if err := atransformer.Initialize(); err != nil {
			spk.logger.Errorf("unable to initilize transformer %v", err)
			return
		}
		spk.textToSpeechTransformer = atransformer
		spk.logger.Benchmark("speak.transformer.Initialize", time.Since(start))
	})

	wg.Wait()
	spk.logger.Benchmark("speak.Init", time.Since(start))
	return nil
}

func (spk *GenericRequestor) OnCompleteSentence(
	ctx context.Context,
	contextId string, output string) error {

	ctx, span, _ := spk.Tracer().StartSpan(spk.Context(), utils.AssistantSpeakingStage)
	defer span.EndSpan(ctx, utils.AssistantSpeakingStage)

	span.AddAttributes(ctx,
		internal_adapter_telemetry.KV{
			K: "messageId", V: internal_adapter_telemetry.StringValue(contextId),
		},
		internal_adapter_telemetry.KV{
			K: "activity", V: internal_adapter_telemetry.StringValue("speak"),
		},
		internal_adapter_telemetry.KV{
			K: "script", V: internal_adapter_telemetry.StringValue(output),
		},
	)
	for _, v := range spk.synthesizers {
		output = v.Synthesize(spk.Context(), contextId, output)
	}
	span.AddAttributes(ctx,
		internal_adapter_telemetry.KV{
			K: "synthesize_script", V: internal_adapter_telemetry.StringValue(output),
		},
	)
	if spk.textToSpeechTransformer != nil {
		spk.textToSpeechTransformer.Transform(spk.Context(), output, &internal_transformer.TextToSpeechOption{
			ContextId:  contextId,
			IsComplete: false,
		})

	}
	return nil
}

func (spk *GenericRequestor) CloseSpeaker() error {
	if spk.textToSpeechTransformer != nil {
		if err := spk.textToSpeechTransformer.Close(spk.Context()); err != nil {
			spk.logger.Errorf("cancel all output transformer with error %v", err)
		}
	}
	return nil
}

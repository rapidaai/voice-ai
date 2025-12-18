// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software.
// Unauthorized copying, modification, or redistribution is strictly prohibited.
package internal_adapter_request_generic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	internal_audio "github.com/rapidaai/api/assistant-api/internal/audio"
	internal_end_of_speech "github.com/rapidaai/api/assistant-api/internal/end_of_speech"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_denoiser_factory "github.com/rapidaai/api/assistant-api/internal/factory/denoiser"
	internal_end_of_speech_factory "github.com/rapidaai/api/assistant-api/internal/factory/end_of_speech"
	internal_adapter_transformer_factory "github.com/rapidaai/api/assistant-api/internal/factory/transformer"
	internal_vad_factory "github.com/rapidaai/api/assistant-api/internal/factory/vad"
	internal_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	internal_transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
	internal_vad "github.com/rapidaai/api/assistant-api/internal/vad"
	"github.com/rapidaai/pkg/utils"
	"golang.org/x/sync/errgroup"
)

func (listening *GenericRequestor) listenTranscript(
	transcript string,
	confidence float64,
	language string,
	isCompleted bool,
) error {
	ctx, span, _ := listening.
		Tracer().StartSpan(
		listening.Context(),
		utils.AssistantListeningStage,
		internal_telemetry.KV{
			K: "transcript",
			V: internal_telemetry.StringValue(transcript),
		}, internal_telemetry.KV{
			K: "confidence",
			V: internal_telemetry.FloatValue(confidence),
		}, internal_telemetry.KV{
			K: "isCompleted",
			V: internal_telemetry.BoolValue(isCompleted),
		})
	defer span.EndSpan(ctx, utils.AssistantListeningStage)

	//
	if transcript != "" {
		_, err := listening.OnRecieveTranscript(
			ctx,
			transcript,
			confidence,
			language,
			isCompleted)
		if err != nil {
			listening.logger.Info("OnRecieveTranscript error %s", err)
		}
		err = listening.ListenText(
			ctx,
			&internal_end_of_speech.STTEndOfSpeechInput{
				Message:    transcript,
				IsComplete: isCompleted,
				Time:       time.Now(),
			})
		if err != nil {
			listening.logger.Info("ListenText error %s", err)
		}
	}
	return nil
}

func (listening *GenericRequestor) initializeSpeechToText(
	ctx context.Context,
	transformerConfig *internal_assistant_entity.AssistantDeploymentAudio,
	audioConfig *internal_audio.AudioConfig,
	options utils.Option) error {
	credentialId, err := options.GetUint64("rapida.credential_id")
	if err != nil {
		listening.logger.Errorf("unable to find credential from options %+v", err)
		return err
	}
	credential, err := listening.
		VaultCaller().
		GetCredential(ctx, listening.Auth(), credentialId)
	if err != nil {
		listening.logger.Errorf("Api call to find credential failed %+v", err)
		return err
	}

	atransformer, err := internal_adapter_transformer_factory.
		GetSpeechToTextTransformer(
			internal_adapter_transformer_factory.AudioTransformer(transformerConfig.AudioProvider),
			listening.Context(), listening.logger, credential,
			&internal_transformer.SpeechToTextInitializeOptions{
				AudioConfig:  audioConfig,
				OnTranscript: listening.listenTranscript,
				ModelOptions: options,
			},
		)

	if err != nil {
		listening.logger.Errorf("unable to create input audio transformer with error %v", err)
		return err
	}
	err = atransformer.Initialize()
	if err != nil {
		listening.logger.Errorf("unable to initilize transformer %v", err)
		return err
	}
	listening.speechToTextTransformer = atransformer
	return nil
}

// Init initializes the audio talking system for a given assistant persona.
// It sets up both audio input and output transformer.
// This function is typically called at the beginning of a communication session.
func (listening *GenericRequestor) ConnectListener(ctx context.Context, audioConfig, audioOutConfig *internal_audio.AudioConfig) error {
	ctx, span, _ := listening.Tracer().StartSpan(ctx, utils.AssistantListenConnectStage)
	defer span.EndSpan(ctx, utils.AssistantListenConnectStage)

	eGroup, ctx := errgroup.WithContext(ctx)
	options := map[string]interface{}{
		"microphone.eos.timeout": 500,
		"microphone.eos.enabled": true,
	}

	transformerConfig, err := listening.GetSpeechToTextTransformer()
	if err != nil {
		listening.logger.Warnf("error during getting transformer for assistant.")
	} else {
		options = utils.MergeMaps(options, transformerConfig.GetOptions())
		span.AddAttributes(ctx,
			internal_telemetry.KV{K: "options", V: internal_telemetry.JSONValue(options)},
			internal_telemetry.KV{K: "provider", V: internal_telemetry.StringValue(transformerConfig.AudioProvider)},
		)
		//
		eGroup.Go(func() error {
			err := listening.initializeSpeechToText(ctx, transformerConfig, audioConfig, options)
			if err != nil {
				listening.logger.Errorf("unable to initialize transformer %+v", err)
			}
			return nil
		})

		eGroup.Go(func() error {
			err := listening.initializeVAD(ctx, audioConfig, options)
			if err != nil {
				listening.logger.Errorf("illegal input audio transformer, check the config and re-init")
			}
			return nil
		})

		eGroup.Go(func() error {
			err := listening.initializeDenoiser(ctx, audioConfig, options)
			if err != nil {
				listening.logger.Errorf("illegal input audio transformer, check the config and re-init")
			}
			return nil
		})

	}

	eGroup.Go(func() error {
		err := listening.initializeEndOfSpeech(ctx, options)
		if err != nil {
			listening.logger.Errorf("illegal input audio transformer, check the config and re-init")
		}
		return nil
	})

	if err := eGroup.Wait(); err != nil {
		return err
	}
	return nil
}

func (listening *GenericRequestor) CloseListener(ctx context.Context) error {
	if listening.speechToTextTransformer != nil {
		if err := listening.speechToTextTransformer.Close(ctx); err != nil {
			listening.logger.Warnf("cancel all output transformer with error %v", err)
		}
	}

	if listening.endOfSpeech != nil {
		if err := listening.endOfSpeech.Close(); err != nil {
			listening.logger.Warnf("cancel end of speech with error %v", err)
		}
	}

	if listening.vad != nil {
		if err := listening.vad.Close(); err != nil {
			listening.logger.Warnf("cancel vad with error %v", err)
		}
	}
	return nil
}

func (listening *GenericRequestor) initializeEndOfSpeech(
	ctx context.Context,
	options utils.Option,
) error {
	start := time.Now()
	provider, err := options.GetString("microphone.eos.provider")
	if err != nil {
		listening.logger.Errorf("denoising.provider is not set, please check the configuration")
		return err
	}
	endOfSpeech, err := internal_end_of_speech_factory.GetEndOfSpeech(
		internal_end_of_speech_factory.EndOfSpeechIdentifier(provider),
		listening.logger,
		func(_ctx context.Context, act *internal_end_of_speech.EndOfSpeechResult) error {
			return listening.afterAnalyze(_ctx, act)
		},
		options)
	if err != nil {
		listening.logger.Warnf("unable to initialize text analyzer %+v", err)
		return err
	}
	listening.endOfSpeech = endOfSpeech
	listening.logger.Benchmark("listen.endOfSpeech", time.Since(start))
	return nil
}

func (listening *GenericRequestor) initializeDenoiser(
	ctx context.Context,
	audioConfig *internal_audio.AudioConfig,
	options utils.Option) error {
	provider, err := options.GetString("microphone.denoising.provider")
	if err != nil {
		listening.logger.Errorf("denoising.provider is not set, please check the configuration")
		return err
	}
	denoise, err := internal_denoiser_factory.GetDenoiser(internal_denoiser_factory.DenoiserIdentifier(provider), listening.logger, audioConfig, options)
	if err != nil {
		listening.logger.Errorf("error wile intializing denoiser %+v", err)
	}
	listening.denoiser = denoise
	return nil
}

func (listening *GenericRequestor) initializeVAD(
	ctx context.Context,
	audioConfig *internal_audio.AudioConfig,
	options utils.Option,
) error {
	start := time.Now()
	provider, err := options.GetString("microphone.vad.provider")
	if err != nil {
		listening.logger.Errorf("vad.provider is not set, please check the configuration")
		return err
	}

	vad, err := internal_vad_factory.GetVAD(
		internal_vad_factory.VADIdentifier(provider),
		listening.logger,
		audioConfig,
		func(vr *internal_vad.VadResult) error {
			return listening.afterAnalyze(listening.Context(), vr)
		},
		options,
	)
	if err != nil {
		listening.logger.Errorf("error wile intializing vad %+v", err)
		return err
	}
	listening.vad = vad
	listening.logger.Benchmark("listen.initializeVAD", time.Since(start))
	return nil
}

func (listening *GenericRequestor) afterAnalyze(
	ctx context.Context,
	a interface{},
) error {
	ctx, span, _ := listening.Tracer().StartSpan(listening.Context(), utils.AssistantUtteranceStage)
	switch v := a.(type) {
	case *internal_end_of_speech.EndOfSpeechResult:
		span.EndSpan(ctx,
			utils.AssistantUtteranceStage,
			internal_telemetry.KV{
				K: "activity_type",
				V: internal_telemetry.StringValue("SpeechEndActivity"),
			},
			internal_telemetry.KV{
				K: "speech_start_at",
				V: internal_telemetry.StringValue(
					time.Unix(int64(v.GetSpeechStartAt()), int64((v.GetSpeechStartAt()-float64(int64(v.GetSpeechStartAt())))*1e9)).
						Format("2006-01-02 15:04:05.000000"),
				),
			},
			internal_telemetry.KV{
				K: "speech_end_at",
				V: internal_telemetry.StringValue(
					time.Unix(int64(v.GetSpeechEndAt()), int64((v.GetSpeechEndAt()-float64(int64(v.GetSpeechEndAt())))*1e9)).
						Format("2006-01-02 15:04:05.000000"),
				),
			},
			internal_telemetry.KV{
				K: "speech",
				V: internal_telemetry.StringValue(v.GetSpeech()),
			},
		)
		return listening.OnSilenceBreak(ctx)
	case *internal_vad.VadResult:
		span.EndSpan(ctx,
			utils.AssistantUtteranceStage,
			internal_telemetry.KV{
				K: "activity_type",
				V: internal_telemetry.StringValue("vad"),
			},
		)
		// might be noise at first
		if v.GetSpeechStartAt() < 3 {
			listening.logger.Warn("interrupt: very early interruption")
			return nil
		}
		listening.OnInterrupt(ctx, "vad")
		listening.ListenText(
			ctx,
			&internal_end_of_speech.
				SystemEndOfSpeechInput{
				Time: time.Now(),
			},
		)
		return nil
	default:
		return fmt.Errorf("unsupported activity type")
	}
}

func (listening *GenericRequestor) ListenText(
	ctx context.Context,
	msg internal_end_of_speech.EndOfSpeechInput) error {
	if listening.endOfSpeech != nil {
		var err error
		utils.Go(ctx, func() {
			err = listening.endOfSpeech.Analyze(ctx, msg)
			if err != nil {
				if err == context.Canceled {
					listening.logger.Info("Analysis canceled due to new content")
				} else {
					listening.logger.Tracef(ctx, "list of analyze text and got an error %+v", err)
				}
			}
		})
		return err
	}
	return listening.OnSilenceBreak(ctx)
}

func (listening *GenericRequestor) ListenAudio(
	ctx context.Context,
	in []byte,
) ([]byte, error) {
	out := in

	if listening.denoiser != nil {
		dnOut, _, err := listening.denoiser.Denoise(ctx, in)
		if err != nil {
			listening.logger.Warnf("error while denoising process | will process actual audio byte")
		}
		out = dnOut
	}
	if listening.vad != nil {
		utils.Go(ctx, func() {
			listening.vad.Process(out)
		})
	}
	if listening.speechToTextTransformer != nil {
		utils.Go(ctx, func() {
			if err := listening.speechToTextTransformer.Transform(ctx, out, nil); err != nil {
				if !errors.Is(err, io.EOF) {
					listening.logger.Tracef(ctx, "error while transforming input %s and error %v", listening.speechToTextTransformer.Name(), err)
				}
			}
		})
	}
	return out, nil
}

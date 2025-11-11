package internal_adapter_request_generic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	internal_analyzers "github.com/rapidaai/api/assistant-api/internal/analyzers"
	internal_analyzer_factories "github.com/rapidaai/api/assistant-api/internal/factories/analyzers"
	internal_adapter_transformer_factories "github.com/rapidaai/api/assistant-api/internal/factories/transformers"
	internal_assistant_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/assistants"
	internal_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	internal_transformers "github.com/rapidaai/api/assistant-api/internal/transformers"
	"github.com/rapidaai/pkg/utils"
)

func (lis *GenericRequestor) ListenAudio(
	ctx context.Context,
	in []byte,
) {
	if lis.speechToTextTransformer != nil {
		for _, v := range lis.audioAnalyzers {
			analyzer := v
			utils.Go(ctx, func() {
				err := analyzer.Analyze(ctx, in)
				if err != nil {
					lis.logger.Warn("analyze issue with error %s", err)
				}
			})
		}
		if err := lis.speechToTextTransformer.Transform(ctx, in, nil); err != nil {
			if !errors.Is(err, io.EOF) {
				lis.logger.Tracef(ctx, "error while transforming input %s and error %v", lis.speechToTextTransformer.Name(), err)
			}
		}
	}
}

func (lis *GenericRequestor) listenTranscript(
	transcript string,
	confidence float64,
	language string,
	isCompleted bool,
) error {
	ctx, span, _ := lis.
		Tracer().StartSpan(
		lis.Context(),
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
		_, err := lis.OnRecieveTranscript(
			ctx,
			transcript,
			confidence,
			language,
			isCompleted)
		if err != nil {
			lis.logger.Info("OnRecieveTranscript error %s", err)
		}
		err = lis.ListenText(
			ctx,
			&internal_analyzers.STTTextAnalyzerInput{
				Message:    transcript,
				IsComplete: isCompleted,
				Time:       time.Now(),
			})
		if err != nil {
			lis.logger.Info("ListenText error %s", err)
		}
	}
	return nil
}

func (lis *GenericRequestor) initSpeechToTextTransformer(
	transformerConfig *internal_assistant_gorm.AssistantDeploymentAudio) error {
	start := time.Now()
	atransformer, err := internal_adapter_transformer_factories.
		GetSpeechToTextTransformer(
			internal_adapter_transformer_factories.
				AudioTransformer(transformerConfig.AudioProvider),
			lis.Context(),
			lis.logger,
			lis,
			&internal_transformers.
				SpeechToTextInitializeOptions{
				AudioConfig:  lis.Streamer().Config().InputConfig.Audio,
				OnTranscript: lis.listenTranscript,
				ModelOptions: utils.
					MergeMaps(
						transformerConfig.
							GetOptions(),
						lis.
							GetOptions(),
					),
			},
		)
	if err != nil {
		lis.logger.Errorf("unable to create input audio transformer with error %v", err)
		return err
	}
	err = atransformer.Initialize()
	if err != nil {
		lis.logger.Errorf("unable to initilize transformer %v", err)
		return err
	}
	lis.speechToTextTransformer = atransformer
	lis.logger.Benchmark("listen.transformer.Initialize", time.Since(start))
	return nil
}

// Init initializes the audio talking system for a given assistant persona.
// It sets up both audio input and output transformers.
// This function is typically called at the beginning of a communication session.
func (lis *GenericRequestor) ConnectListener(ctx context.Context) error {
	ctx, span, _ := lis.Tracer().StartSpan(ctx, utils.AssistantListenConnectStage)
	defer span.EndSpan(ctx, utils.AssistantListenConnectStage)

	start := time.Now()
	defer lis.logger.Benchmark("listen.Init", time.Since(start))
	var wg sync.WaitGroup
	options := map[string]interface{}{
		"microphone.eos.timeout": 500,
		"microphone.eos.enabled": true,
	}
	transformerConfig, err := lis.
		GetSpeechToTextTransformer()
	if err == nil {
		options = utils.MergeMaps(
			options,
			transformerConfig.GetOptions())
		span.AddAttributes(ctx,
			internal_telemetry.KV{K: "options", V: internal_telemetry.JSONValue(options)},
			internal_telemetry.KV{K: "provider", V: internal_telemetry.StringValue(transformerConfig.AudioProvider)},
		)

		wg.Add(1)
		utils.Go(ctx, func() {
			defer wg.Done()
			lis.initSpeechToTextTransformer(transformerConfig)
		})

		wg.Add(1)
		utils.Go(ctx, func() {
			defer wg.Done()
			err := lis.initVoiceUtteranceStartAnalyzer(ctx, transformerConfig)
			if err != nil {
				lis.logger.Errorf("illegal input audio transformer, check the config and re-init")
			}
		})

	}

	wg.Add(1)
	utils.Go(ctx, func() {
		defer wg.Done()
		err := lis.initTextUtteranceEndAnalyzer(ctx, options)
		if err != nil {
			lis.logger.Errorf("illegal input audio transformer, check the config and re-init")
		}

	})

	wg.Wait()
	return nil
}

func (lis *GenericRequestor) CloseListener(ctx context.Context) error {
	if lis.speechToTextTransformer != nil {
		if err := lis.speechToTextTransformer.Close(ctx); err != nil {
			lis.logger.Warnf("cancel all output transformer with error %v", err)
		}
	}
	for _, v := range lis.audioAnalyzers {
		err := v.Close()
		if err != nil {
			lis.logger.Warnf("error while canceling audio analyzers error %v", err)
		}
	}
	for _, v := range lis.textAnalyzers {
		err := v.Close()
		if err != nil {
			lis.logger.Warnf("error while canceling audio analyzers error %v", err)
		}
	}
	return nil
}

func (lis *GenericRequestor) initTextUtteranceEndAnalyzer(
	ctx context.Context,
	options utils.Option,
) error {
	start := time.Now()
	opts := &internal_analyzers.TextAnalyzerOptions{
		OnAnalyze: func(_ctx context.Context, act internal_analyzers.Activity) error {
			return lis.afterAnalyze(_ctx, act)
		},
	}
	an, err := internal_analyzer_factories.GetTextAnalyzer(
		internal_analyzer_factories.UtteranceEndAnalyzer, lis.logger,
		opts.WithOptions(
			utils.MergeMaps(
				options,
				lis.GetOptions(),
			),
		))
	if err != nil {
		lis.logger.Warnf("unable to initialize text analyzer %+v", err)
		return err
	}
	lis.textAnalyzers = append(lis.textAnalyzers, an)
	lis.logger.Benchmark("listen.InitTextUtteranceEndAnalyzer", time.Since(start))
	return nil
}

func (lis *GenericRequestor) initVoiceUtteranceStartAnalyzer(
	ctx context.Context,
	transformerConfig *internal_assistant_gorm.AssistantDeploymentAudio,
) error {
	start := time.Now()
	opts := &internal_analyzers.VoiceAnalyzerOptions{
		OnAnalyze: func(ctx context.Context, act internal_analyzers.Activity) error {
			return lis.afterAnalyze(ctx, act)
		},
	}

	an, err := internal_analyzer_factories.GetVoiceAnalyzer(
		internal_analyzer_factories.UtteranceStartAnalyzer,
		lis.logger,
		lis.Streamer().Config().InputConfig.Audio,
		opts.WithOptions(
			utils.MergeMaps(
				transformerConfig.GetOptions(),
				lis.GetOptions(),
			),
		))
	if err != nil {
		lis.logger.Warnf("unable to initialize vad %+v", err)
		return err
	}
	lis.audioAnalyzers = append(lis.audioAnalyzers, an)
	lis.logger.Benchmark("listen.InitVoiceUtteranceStartAnalyzer", time.Since(start))
	return nil
}

func (lis *GenericRequestor) ListenText(
	ctx context.Context,
	msg internal_analyzers.TextAnalyzerInput) error {
	var err error
	for _, v := range lis.textAnalyzers {
		analyzer := v
		utils.Go(ctx, func() {
			err = analyzer.Analyze(ctx, msg)
			if err != nil {
				if err == context.Canceled {
					lis.logger.Info("Analysis canceled due to new content")
				} else {
					lis.logger.Tracef(ctx, "list of analyze text and got an error %+v", err)
				}
			}
		})
	}
	return err
}

func (lis *GenericRequestor) afterAnalyze(
	ctx context.Context,
	a internal_analyzers.Activity,
) error {
	ctx, span, _ := lis.Tracer().StartSpan(lis.Context(), utils.AssistantUtteranceStage)

	//
	switch v := a.(type) {
	case internal_analyzers.SpeechStartActivity:
		span.EndSpan(ctx,
			utils.AssistantUtteranceStage,
			internal_telemetry.KV{
				K: "activity_type",
				V: internal_telemetry.StringValue("vad"),
			},
			internal_telemetry.KV{
				K: "energy",
				V: internal_telemetry.FloatValue(v.GetEnergy()),
			},
			internal_telemetry.KV{
				K: "confidence",
				V: internal_telemetry.FloatValue(v.GetConfidence()),
			},
		)
		if v.GetSpeechStartAt() < 1 {
			lis.logger.Warn("interrupt: very early interruption")
			return nil
		}

		lis.ListenText(
			ctx,
			&internal_analyzers.SystemTextAnalyzerInput{
				Time: time.Now(),
			},
		)
		lis.OnInterrupt(ctx, "vad")
		lis.ListenText(
			ctx,
			&internal_analyzers.
				SystemTextAnalyzerInput{
				Time: time.Now(),
			},
		)
		return nil
	case internal_analyzers.SpeechEndActivity:
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
		return lis.OnSilenceBreak(ctx, v)

	default:
		return fmt.Errorf("unsupported activity type")
	}
}

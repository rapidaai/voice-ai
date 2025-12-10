package internal_adapter_request_talking_debugger

import (
	"context"
	"fmt"
	"sync"
	"time"

	internal_adapter_requests "github.com/rapidaai/api/assistant-api/internal/adapters/requests"
	internal_adapter_request_customizers "github.com/rapidaai/api/assistant-api/internal/adapters/requests/customizers"
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_telemetry "github.com/rapidaai/api/assistant-api/internal/telemetry"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (talking *debuggerTalking) Disconnect() {
	ctx, span, _ := talking.
		Tracer().
		StartSpan(talking.Context(), utils.AssistantDisconnectStage)

	start := time.Now()
	var wg sync.WaitGroup
	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		metrics := []*types.Metric{
			{
				Name:        type_enums.TIME_TAKEN.String(),
				Value:       fmt.Sprintf("%d", int64(time.Since(talking.StartedAt))),
				Description: "Time taken to complete conversation",
			},
			{
				Name:        type_enums.STATUS.String(),
				Value:       type_enums.RECORD_COMPLETE.String(),
				Description: "Status of the given conversation",
			},
		}
		talking.AddMetrics(talking.Auth(), metrics...)
	})
	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		err := talking.
			CloseListener(ctx)
		if err != nil {
			talking.logger.Tracef(ctx, "unable to cleanup input transformer with error %+v", err)
			return
		}
	})
	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		err := talking.
			CloseSpeaker()
		if err != nil {
			talking.logger.Tracef(ctx, "unable to cleanup input transformer with error %+v", err)
			return
		}
	})
	wg.Wait()
	talking.
		OnEndConversation()
	// you do not need to wait for the recorder
	utils.Go(talking.Context(), func() {
		byt, err := talking.Recorder().Persist()
		if err != nil {
			talking.logger.Tracef(ctx, "unable to persist the audio %+v", err)
			return
		}
		err = talking.CreateConversationRecording(byt)
		if err != nil {
			talking.logger.Tracef(ctx, "unable to create conversation recording %+v", err)
			return
		}
	})
	span.EndSpan(ctx, utils.AssistantDisconnectStage)
	talking.Tracer().Export(ctx,
		talking.Auth(),
		&internal_telemetry.VoiceAgentExportOption{
			AssistantId:              talking.Assistant().Id,
			AssistantProviderModelId: talking.Assistant().AssistantProviderId,
			AssistantConversationId:  talking.Conversation().Id,
		},
	)
	talking.AssistantExecutor().Disconnect(
		ctx,
		talking.Assistant().Id,
		talking.Conversation().Id,
	)
	talking.logger.Benchmark("talking.OnEndSession", time.Since(start))
}

func (talking *debuggerTalking) Connect(
	ctx context.Context,
	iAuth types.SimplePrinciple,
	identifier string,
	req *protos.AssistantConversationConfiguration) error {
	ctx, span, err := talking.
		Tracer().StartSpan(
		ctx,
		utils.AssistantConnectStage,
	)
	defer span.EndSpan(ctx, utils.AssistantConnectStage)

	customization, err := internal_adapter_request_customizers.
		NewRequestBaseCustomizer(req)

	if err != nil {
		talking.logger.Errorf("unable to initialize customizer %+v", err)
		return err
	}
	talking.
		SetAuth(iAuth)

	assistant, err := talking.
		GetAssistant(iAuth,
			req.Assistant.AssistantId,
			req.Assistant.Version)

	if err != nil {
		talking.logger.Errorf("unable to initialize assistant %+v", err)
		return err
	}

	if req.GetAssistantConversationId() > 0 {
		span.AddAttributes(ctx, internal_telemetry.KV{
			K: "conversation_initiation",
			V: internal_telemetry.StringValue("resume"),
		}, internal_telemetry.KV{
			K: "conversation_id",
			V: internal_telemetry.IntValue(req.GetAssistantConversationId()),
		})
		return talking.OnResumeSession(ctx, assistant, identifier, req.GetAssistantConversationId(), customization)
	}
	span.AddAttributes(ctx, internal_telemetry.KV{
		K: "conversation_initiation",
		V: internal_telemetry.StringValue("new"),
	})
	return talking.OnCreateSession(ctx, assistant, identifier, customization)
}

func (talking *debuggerTalking) OnCreateSession(
	ctx context.Context,
	assistant *internal_assistant_entity.Assistant,
	identifier string,
	customization internal_adapter_requests.Customization,
) error {

	ctx, span, err := talking.
		Tracer().StartSpan(
		ctx,
		utils.AssistantCreateConversationStage,
	)
	defer span.EndSpan(ctx, utils.AssistantCreateConversationStage)
	streamInConfg, err := talking.Streamer().Config().GetInputConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	audioInConfig, err := streamInConfg.GetAudioConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	streamOtConfg, err := talking.Streamer().Config().GetOutputConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	audioOutConfig, err := streamOtConfg.GetAudioConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	// establish listener
	utils.Go(ctx, func() {
		defer wg.Done()
		err := talking.ConnectListener(ctx, audioInConfig, audioOutConfig)
		if err != nil {
			talking.logger.Tracef(ctx, "unable to init analyzer %+v", err)
		}
	})

	wg.Add(1)
	utils.Go(ctx, func() {
		defer wg.Done()
		err := talking.ConnectSpeaker(ctx, audioInConfig, audioOutConfig)
		if err != nil {
			talking.logger.Tracef(ctx, "unable to connect speaker %+v", err)
		}

	})

	wg.Add(1)
	utils.Go(ctx, func() {
		defer wg.Done()
		err := talking.Recorder().
			Init(audioInConfig, audioOutConfig)
		if err != nil {
			talking.logger.Tracef(ctx, "unable to init recorder %+v", err)
		}
	})

	// do the conversation
	conversation, err := talking.
		BeginConversation(
			talking.Auth(),
			assistant,
			type_enums.DIRECTION_INBOUND,
			identifier,
			customization.GetArgs(),
			customization.GetMetadata(),
			customization.GetOptions())
	if err != nil {
		talking.logger.Errorf("unable to begin convsersation %+v", err)
		return err
	}
	err = talking.
		Notify(
			ctx,
			&protos.AssistantConversationConfiguration{
				AssistantConversationId: conversation.Id,
				Assistant: &protos.AssistantDefinition{
					AssistantId: assistant.Id,
					Version:     fmt.Sprintf("vrsn_%d", assistant.AssistantProviderId),
				},
				Time: timestamppb.Now(),
			},
		)
	if err != nil {
		talking.logger.Errorf("Error sending configuration: %v\n", err)
	}

	wg.Add(1)
	utils.Go(ctx, func() {
		defer wg.Done()
		talking.logger.Debugf("talking.OnStartSession.executor.Init")
		talking.
			AssistantExecutor().
			Init(
				ctx,
				talking,
			)
	})

	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		talking.AddMetrics(
			talking.Auth(),
			&types.Metric{
				Name:        type_enums.STATUS.String(),
				Value:       type_enums.RECORD_IN_PROGRESS.String(),
				Description: "Status of the given conversation",
			})
	})

	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		client := types.
			GetClientInfoFromGrpcContext(ctx)
		if client != nil {
			clj, _ := client.ToJson()
			talking.SetMetadata(talking.Auth(), map[string]interface{}{
				"talk.client_information": clj,
			})
		}
	})

	wg.Wait()
	err = talking.Greeting(ctx)
	if err != nil {
		talking.logger.Errorf("unable to greet user with error %+v", err)
	}
	talking.
		AssistantExecutor().
		Connect(ctx, talking.Assistant().Id, talking.Conversation().Id)
	err = talking.OnBeginConversation()
	if err != nil {
		talking.logger.Errorf("unable to call hook for begin conversation with error %+v", err)
	}

	return nil
}

func (talking *debuggerTalking) OnResumeSession(
	ctx context.Context,
	assistant *internal_assistant_entity.Assistant,
	identifier string,
	assistantConversationId uint64,
	customization internal_adapter_requests.Customization) error {
	ctx, span, err := talking.
		Tracer().StartSpan(
		talking.Context(),
		utils.AssistantResumeConverstaionStage,
	)
	defer span.EndSpan(ctx, utils.AssistantResumeConverstaionStage)

	streamInConfg, err := talking.Streamer().Config().GetInputConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	audioInConfig, err := streamInConfg.GetAudioConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	streamOtConfg, err := talking.Streamer().Config().GetOutputConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	audioOutConfig, err := streamOtConfg.GetAudioConfig()
	if err != nil {
		talking.logger.Errorf("streamConfg is not set, please check the configuration")
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	// establish listener
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		err := talking.ConnectListener(ctx, audioInConfig, audioOutConfig)
		if err != nil {
			talking.logger.Tracef(talking.Context(), "unable to init analyzer %+v", err)
		}
	})
	// establish speaker

	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		err := talking.ConnectSpeaker(ctx, audioInConfig, audioOutConfig)
		if err != nil {
			talking.logger.Tracef(talking.Context(), "unable to init analyzer %+v", err)
		}

	})

	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		err := talking.Recorder().Init(
			audioInConfig,
			audioOutConfig,
		)
		if err != nil {
			talking.logger.Tracef(talking.Context(), "unable to init recorder %+v", err)
		}
	})

	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		client := types.GetClientInfoFromGrpcContext(talking.Context())
		if client != nil {
			clj, _ := client.ToJson()
			talking.SetMetadata(talking.Auth(), map[string]interface{}{
				"talk.client_information": clj,
			})
		}
	})

	// resume the conversation
	conversation, err := talking.ResumeConversation(
		talking.Auth(),
		assistant,
		assistantConversationId,
		identifier)
	if err != nil {
		talking.logger.Errorf("unable to resume convsersation %+v", err)
		return err
	}
	err = talking.
		Notify(
			ctx,
			&protos.AssistantConversationConfiguration{
				AssistantConversationId: conversation.Id,
				Assistant: &protos.AssistantDefinition{
					AssistantId: assistant.Id,
					Version:     fmt.Sprintf("vrsn_%d", assistant.AssistantProviderId),
				},
				Time: timestamppb.Now(),
			},
		)
	if err != nil {
		talking.logger.Errorf("Error sending configuration: %v\n", err)
	}

	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		talking.logger.Debugf("talking.OnStartSession.executor.Init")
		talking.
			AssistantExecutor().
			Init(
				ctx,
				talking,
			)
	})

	wg.Add(1)
	utils.Go(talking.Context(), func() {
		defer wg.Done()
		talking.AddMetrics(
			talking.Auth(),
			&types.Metric{
				Name:        type_enums.STATUS.String(),
				Value:       type_enums.RECORD_IN_PROGRESS.String(),
				Description: "Status of the given conversation",
			})
	})
	wg.Wait()

	err = talking.Greeting(ctx)
	if err != nil {
		talking.logger.Errorf("unable to greet user with error %+v", err)
	}
	talking.
		AssistantExecutor().
		Connect(ctx, talking.Assistant().Id, talking.Conversation().Id)
	talking.OnResumeConversation()
	return nil
}

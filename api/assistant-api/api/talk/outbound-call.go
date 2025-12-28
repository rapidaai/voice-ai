// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_talk_api

import (
	"context"
	"fmt"

	internal_factories "github.com/rapidaai/api/assistant-api/internal/factory"
	telephony "github.com/rapidaai/api/assistant-api/internal/factory/telephony"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// InitiateAssistantTalk implements protos.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreatePhoneCall(ctx context.Context, ir *protos.CreatePhoneCallRequest) (*protos.CreatePhoneCallResponse, error) {
	auth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		cApi.logger.Errorf("unable to resolve the authentication object, please check the parameter for authentication")
		return utils.AuthenticateError[protos.CreatePhoneCallResponse]()
	}

	toNumber := ir.GetToNumber()
	if utils.IsEmpty(toNumber) {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, fmt.Errorf("missing to_phone parameter"), "Please provide the required to_phone parameter.")
	}

	mtd, err := utils.AnyMapToInterfaceMap(ir.GetMetadata())
	if err != nil {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Illegal metadata for initialize request, please check and try again.")
	}

	args, err := utils.AnyMapToInterfaceMap(ir.GetArgs())
	if err != nil {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Illegal options for initialize request, please check and try again.")
	}

	opts, err := utils.AnyMapToInterfaceMap(ir.GetOptions())
	if err != nil {
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Illegal arguments for initialize request, please check and try again.")
	}

	assistant, err := cApi.assistantService.Get(ctx,
		auth,
		ir.GetAssistant().GetAssistantId(),
		utils.GetVersionDefinition(ir.GetAssistant().GetVersion()),
		&internal_services.GetAssistantOption{
			InjectPhoneDeployment: true,
		})
	if err != nil {
		cApi.logger.Debugf("illegal unable to find assistant %v", err)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Invalid assistant id, please check and try again.")
	}

	if !assistant.IsPhoneDeploymentEnable() {
		cApi.logger.Debugf("illegal deployment for phone %v", err)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Phone deployment not enabled or incomplete, please check rapida console and update the deployment")
	}

	// creating conversation
	conversation, err := cApi.assistantConversationService.
		CreateConversation(
			ctx,
			auth,
			internal_factories.Identifier(utils.PhoneCall, ctx, auth, toNumber),
			assistant.Id,
			assistant.AssistantProviderId,
			type_enums.DIRECTION_OUTBOUND,
			utils.PhoneCall,
		)
	if err != nil {
		cApi.logger.Errorf("unable to create conversation %+v", err)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Unable to create conversation session, please check and try again.")
	}
	o, err := cApi.assistantConversationService.
		ApplyConversationOption(
			ctx, auth, assistant.Id, conversation.Id, opts,
		)
	if err != nil {
		cApi.logger.Debugf("unable to create options %v", err)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Unable to create conversation options, please check and try again.")
	}
	conversation.Options = o
	// updating arguments
	arguments, err := cApi.assistantConversationService.
		ApplyConversationArgument(
			ctx, auth, assistant.Id, conversation.Id, args,
		)
	if err != nil {
		cApi.logger.Debugf("unable to create argument %v", err)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Unable to create conversation arguments, please check and try again.")
	}
	conversation.Arguments = arguments
	// updating metadata
	metadatas, err := cApi.assistantConversationService.ApplyConversationMetadata(
		ctx, auth, assistant.Id, conversation.Id,
		types.NewMetadataList(mtd),
	)
	if err != nil {
		cApi.logger.Debugf("unable to create metadatas %v", err)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Unable to create conversation metadata, please check and try again.")
	}
	conversation.Metadatas = metadatas
	credentialID, err := assistant.
		AssistantPhoneDeployment.
		GetOptions().
		GetUint64("rapida.credential_id")
	if err != nil {
		cApi.
			assistantConversationService.
			ApplyConversationMetrics(
				ctx,
				auth,
				assistant.Id,
				conversation.Id,
				[]*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
			)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Please check the credential for telephony, please check and try again.")
	}
	vltC, err := cApi.vaultClient.GetCredential(ctx, auth, credentialID)
	if err != nil {
		cApi.assistantConversationService.
			ApplyConversationMetrics(
				ctx, auth, assistant.Id, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
			)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Please check the credential for telephony, please check and try again.")
	}

	telephony, err := telephony.GetTelephony(
		telephony.Telephony(
			assistant.
				AssistantPhoneDeployment.
				TelephonyProvider),
		cApi.cfg,
		cApi.logger,
	)
	if err != nil {
		cApi.
			assistantConversationService.
			ApplyConversationMetrics(
				ctx, auth, assistant.Id, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
			)
		return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, err, "Please check the configuration for telephony, please check and try again.")
	}

	fromPhone := ir.GetFromNumber()
	if utils.IsEmpty(fromPhone) {
		fromNumber, err := assistant.
			AssistantPhoneDeployment.
			GetOptions().
			GetString("phone")
		if err != nil {
			cApi.assistantConversationService.
				ApplyConversationMetrics(
					ctx, auth, assistant.Id, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
				)
			return utils.ErrorWithCode[protos.CreatePhoneCallResponse](200, fmt.Errorf("failed to get Twilio phone number"), "Unable to retrieve the default phone number.")
		}
		fromPhone = fromNumber
	}

	meta, metric, event, err := telephony.MakeCall(
		auth,
		toNumber,
		fromPhone,
		ir.GetAssistant().GetAssistantId(),
		conversation.Id,
		vltC,
		assistant.
			AssistantPhoneDeployment.
			GetOptions(),
	)
	if err != nil {
		cApi.logger.Errorf("telephony call return error %v", err)
	}

	if metric != nil {
		metrics, err := cApi.assistantConversationService.
			ApplyConversationMetrics(
				ctx, auth, assistant.Id, conversation.Id, metric,
			)
		if err == nil {
			conversation.Metrics = append(conversation.Metrics, metrics...)
		}
	}
	if meta != nil {
		mtds, err := cApi.assistantConversationService.
			ApplyConversationMetadata(
				ctx, auth, assistant.Id, conversation.Id, meta,
			)
		if err == nil {
			conversation.Metadatas = append(conversation.Metadatas, mtds...)
		}
	}
	if event != nil {
		evts, err := cApi.assistantConversationService.
			ApplyConversationTelephonyEvent(
				ctx, auth, assistant.AssistantPhoneDeployment.TelephonyProvider, assistant.Id, conversation.Id, event,
			)
		if err == nil {
			conversation.TelephonyEvents = append(conversation.TelephonyEvents, evts...)
		}
	}

	out := &protos.AssistantConversation{}
	err = utils.Cast(conversation, out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant conversation %v", err)
	}
	return utils.Success[protos.CreatePhoneCallResponse, *protos.AssistantConversation](out)
}

// InitiateBulkAssistantTalk implements protos.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreateBulkPhoneCall(ctx context.Context, ir *protos.CreateBulkPhoneCallRequest) (*protos.CreateBulkPhoneCallResponse, error) {
	_, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		cApi.logger.Errorf("unable to resolve the authentication object, please check the parameter for authentication")
		return utils.AuthenticateError[protos.CreateBulkPhoneCallResponse]()
	}

	out := make([]*protos.AssistantConversation, 0)
	for _, v := range ir.GetPhoneCalls() {
		resp, err := cApi.CreatePhoneCall(ctx, v)
		if err != nil {
			cApi.logger.Errorf("error while making call %+v", err)
		}
		if resp.GetData() != nil {
			out = append(out, resp.GetData())
		}

	}
	return utils.Success[protos.CreateBulkPhoneCallResponse, []*protos.AssistantConversation](out)
}

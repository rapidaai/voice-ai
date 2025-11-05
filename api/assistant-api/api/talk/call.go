package assistant_talk_api

import (
	"context"
	"fmt"

	internal_factories "github.com/rapidaai/api/assistant-api/internal/factories"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	lexatic_backend "github.com/rapidaai/protos"
)

// InitiateAssistantTalk implements lexatic_backend.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreatePhoneCall(ctx context.Context, ir *lexatic_backend.CreatePhoneCallRequest) (*lexatic_backend.CreatePhoneCallResponse, error) {
	auth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		cApi.logger.Errorf("unable to resolve the authentication object, please check the parameter for authentication")
		return utils.AuthenticateError[lexatic_backend.CreatePhoneCallResponse]()
	}

	toNumber := ir.GetToNumber()
	if utils.IsEmpty(toNumber) {
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, fmt.Errorf("missing to_phone parameter"), "Please provide the required to_phone parameter.")
	}

	mtd, err := utils.AnyMapToInterfaceMap(ir.GetMetadata())
	if err != nil {
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Illegal metadata for initialize request, please check and try again.")
	}

	args, err := utils.AnyMapToInterfaceMap(ir.GetArgs())
	if err != nil {
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Illegal options for initialize request, please check and try again.")
	}

	opts, err := utils.AnyMapToInterfaceMap(ir.GetOptions())
	if err != nil {
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Illegal arguments for initialize request, please check and try again.")
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
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Invalid assistant id, please check and try again.")
	}

	if !assistant.IsPhoneDeploymentEnable() {
		cApi.logger.Debugf("illegal deployment for phone %v", err)
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Phone deployment not enabled or incomplete, please check rapida console and update the deployment")
	}

	// creating conversation
	conversation, err := cApi.assistantConversationService.CreateConversation(
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
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Unable to create conversation session, please check and try again.")
	}
	o, err := cApi.assistantConversationService.ApplyConversationOption(
		ctx, auth, conversation.Id, opts,
	)
	if err != nil {
		cApi.logger.Debugf("unable to create options %v", err)
	}
	conversation.Options = o

	// updating arguments
	arguments, err := cApi.assistantConversationService.ApplyConversationArgument(
		ctx, auth, conversation.Id, args,
	)
	if err != nil {
		cApi.logger.Debugf("unable to create argument %v", err)
	}
	conversation.Arguments = arguments

	// updating metadata
	metadatas, err := cApi.assistantConversationService.ApplyConversationMetadata(
		ctx, auth, conversation.Id,
		mtd,
	)
	if err != nil {
		cApi.logger.Debugf("unable to create metadatas %v", err)
	}
	conversation.Metadatas = metadatas

	//
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
				conversation.Id,
				[]*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
			)
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Please check the credential for telephony, please check and try again.")
	}

	vltC, err := cApi.vaultClient.GetCredential(ctx, auth, credentialID)
	if err != nil {
		cApi.assistantConversationService.
			ApplyConversationMetrics(
				ctx, auth, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
			)
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Please check the credential for telephony, please check and try again.")
	}

	telephony, err := internal_factories.GetTelephony(
		internal_factories.Telephony(assistant.
			AssistantPhoneDeployment.
			TelephonyProvider),
		cApi.logger,
		vltC,
		assistant.
			AssistantPhoneDeployment.
			GetOptions(),
	)
	if err != nil {
		cApi.
			assistantConversationService.
			ApplyConversationMetrics(
				ctx, auth, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
			)
		return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, err, "Please check the configuration for telephony, please check and try again.")
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
					ctx, auth, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
				)
			return utils.ErrorWithCode[lexatic_backend.CreatePhoneCallResponse](200, fmt.Errorf("failed to get Twilio phone number"), "Unable to retrieve the default phone number.")
		}
		fromPhone = fromNumber
	}
	mtd["talk.client"] = map[string]interface{}{
		"to_phone":   toNumber,
		"from_phone": fromPhone,
	}

	outcome, err := telephony.CreateCall(
		auth,
		toNumber,
		fromPhone,
		ir.GetAssistant().GetAssistantId(),
		conversation.Id,
	)
	if err != nil {
		metrics, mErr := cApi.assistantConversationService.
			ApplyConversationMetrics(
				ctx,
				auth,
				conversation.Id,
				[]*types.Metric{types.NewStatusMetric(type_enums.RECORD_FAILED)},
			)
		if mErr == nil {
			conversation.Metrics = append(conversation.Metrics, metrics...)
		}
		mtd = utils.
			MergeMaps(mtd, map[string]interface{}{"talk.outgoing_api_response_error": err.Error()})
	}
	//
	if outcome != nil {
		metrics, err := cApi.assistantConversationService.
			ApplyConversationMetrics(
				ctx, auth, conversation.Id, []*types.Metric{types.NewStatusMetric(type_enums.RECORD_QUEUED)},
			)
		if err == nil {
			conversation.Metrics = append(conversation.Metrics, metrics...)
		}
		mtd = utils.MergeMaps(mtd, map[string]interface{}{"talk.outgoing_api_response": outcome})

	}

	// updating metadata
	metadatas, err = cApi.assistantConversationService.ApplyConversationMetadata(
		ctx, auth, conversation.Id,
		mtd,
	)
	if err == nil {
		conversation.Metadatas = append(conversation.Metadatas, metadatas...)
	}

	// output
	out := &lexatic_backend.AssistantConversation{}
	err = utils.Cast(conversation, out)
	if err != nil {
		cApi.logger.Errorf("unable to cast assistant conversation %v", err)
	}
	return utils.Success[lexatic_backend.CreatePhoneCallResponse, *lexatic_backend.AssistantConversation](out)
}

// InitiateBulkAssistantTalk implements lexatic_backend.TalkServiceServer.
func (cApi *ConversationGrpcApi) CreateBulkPhoneCall(ctx context.Context, ir *lexatic_backend.CreateBulkPhoneCallRequest) (*lexatic_backend.CreateBulkPhoneCallResponse, error) {
	_, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated {
		cApi.logger.Errorf("unable to resolve the authentication object, please check the parameter for authentication")
		return utils.AuthenticateError[lexatic_backend.CreateBulkPhoneCallResponse]()
	}

	out := make([]*lexatic_backend.AssistantConversation, 0)
	for _, v := range ir.GetPhoneCalls() {
		resp, err := cApi.CreatePhoneCall(ctx, v)
		if err != nil {
			cApi.logger.Errorf("error while making call %+v", err)
		}
		if resp.GetData() != nil {
			out = append(out, resp.GetData())
		}

	}
	return utils.Success[lexatic_backend.CreateBulkPhoneCallResponse, []*lexatic_backend.AssistantConversation](out)
}

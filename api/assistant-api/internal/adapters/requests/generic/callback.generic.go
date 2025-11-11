package internal_adapter_request_generic

import (
	"context"

	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

/*
*  Generation of text from the executors
*  default executor or remote executor
 */
func (talking *GenericRequestor) OnGenerationComplete(
	ctx context.Context,
	messageid string,
	ouput *types.Message,
	metrics []*types.Metric) error {
	//
	err := talking.Output(ctx, messageid, ouput, true)
	if err != nil {
		talking.logger.Errorf("unable to output text for the message %s", messageid)
	}

	bCtx := talking.Context()
	utils.Go(bCtx, func() {
		err := talking.OnUpdateMessage(
			talking.Context(),
			messageid,
			ouput,
			type_enums.RECORD_COMPLETE)
		if err != nil {
			talking.logger.Errorf("Error in OnUpdateMessage: %v", err)
		}

		if ouput.Meta != nil {
			talking.OnMessageMetadata(bCtx, messageid, ouput.Meta)
		}

	})
	utils.Go(bCtx, func() {
		if err := talking.OnMessageMetric(bCtx, messageid, metrics); err != nil {
			talking.logger.Errorf("Error in OnUpdateMessage: %v", err)
		}
	})
	return nil
}

/**/
func (talking *GenericRequestor) OnGeneration(
	ctx context.Context,
	messageid string,
	out *types.Message) error {

	return talking.Output(
		ctx,
		messageid,
		out,
		false)
}

func (talking *GenericRequestor) Execute(
	ctx context.Context,
	messageid string, in *types.Message) error {
	in = talking.OnRecieveMessage(in)
	utils.Go(ctx, func() {
		talking.OnCreateMessage(
			ctx,
			messageid,
			in,
		)
	})
	utils.Go(ctx, func() {
		talking.OnMessageMetadata(
			ctx,
			messageid,
			in.Meta,
		)
	})
	err := talking.assistantExecutor.Talk(ctx, messageid, in, talking)
	if err != nil {
		msg, err := talking.GetBehavior()
		if err != nil {
			talking.logger.Warnf("no on error message setup for assistant.")
			return nil
		}
		if msg.Mistake != nil {
			return talking.Output(ctx, messageid, &types.Message{
				Role: "assistant",
				Contents: []*types.Content{{
					ContentType:   commons.TEXT_CONTENT.String(),
					ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
					Content:       []byte(*msg.Mistake),
				}}}, true)
		}
		return talking.Output(ctx, messageid, &types.Message{
			Role: "assistant",
			Contents: []*types.Content{{
				ContentType:   commons.TEXT_CONTENT.String(),
				ContentFormat: commons.TEXT_CONTENT_FORMAT_RAW.String(),
				Content:       []byte("Oops! It looks like something went wrong. Let me look into that for you right away. I really appreciate your patience—hang tight while I get this sorted!"),
			}}}, true)
	}
	return nil
}

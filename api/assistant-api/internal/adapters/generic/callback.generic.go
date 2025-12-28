// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_adapter_request_generic

import (
	"context"

	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
)

/*
*  Generation of text from the executors
*  default executor or remote executor
 */
func (talking *GenericRequestor) OnGenerationComplete(ctx context.Context, messageid string, ouput *types.Message, metrics []*types.Metric) error {
	if err := talking.Output(ctx, messageid, ouput, true); err != nil {
		talking.logger.Errorf("unable to output text for the message %s", messageid)
	}
	utils.Go(talking.Context(), func() {
		if err := talking.OnUpdateMessage(talking.Context(), messageid, ouput, type_enums.RECORD_COMPLETE); err != nil {
			talking.logger.Errorf("Error in OnUpdateMessage: %v", err)
		}
		if ouput.Meta != nil {
			talking.OnMessageMetadata(talking.Context(), messageid, ouput.Meta)
		}

	})
	utils.Go(talking.Context(), func() {
		// if there are metrics generated from the generation, we need to log them
		if len(metrics) > 0 {
			if err := talking.OnMessageMetric(talking.Context(), messageid, metrics); err != nil {
				talking.logger.Errorf("Error in OnUpdateMessage: %v", err)
			}
		}
	})
	return nil
}

/**/
func (talking *GenericRequestor) OnGeneration(ctx context.Context, messageid string, out *types.Message) error {
	return talking.Output(ctx, messageid, out, false)
}

func (talking *GenericRequestor) Execute(ctx context.Context, messageid string, in *types.Message) error {
	in = talking.OnRecieveMessage(in)
	utils.Go(ctx, func() {
		if err := talking.OnCreateMessage(ctx, messageid, in); err != nil {
			talking.logger.Errorf("Error in OnCreateMessage: %v", err)
		}
	})
	utils.Go(ctx, func() {
		if err := talking.OnMessageMetadata(ctx, messageid, in.Meta); err != nil {
			talking.logger.Errorf("Error in OnMessageMetadata: %v", err)
		}
	})
	if err := talking.assistantExecutor.Talk(ctx, messageid, in, talking); err != nil {
		talking.OnError(ctx, messageid)
		return nil
	}
	return nil
}

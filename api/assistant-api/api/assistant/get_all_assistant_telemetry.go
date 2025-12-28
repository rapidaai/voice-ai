// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package assistant_api

import (
	"context"

	internal_assistant_telemetry_exporters "github.com/rapidaai/api/assistant-api/internal/telemetry/assistant/exporters"
	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
)

// GetAllAssistantConversationTelemetry implements protos.AssistantServiceServer.
func (assistantApi *assistantGrpcApi) GetAllAssistantTelemetry(ctx context.Context, request *protos.GetAllAssistantTelemetryRequest) (*protos.GetAllAssistantTelemetryResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		assistantApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[protos.GetAllAssistantTelemetryResponse]()
	}

	otelExporter := internal_assistant_telemetry_exporters.NewOpensearchAssistantTraceExporter(
		assistantApi.logger,
		&assistantApi.cfg.AppConfig,
		assistantApi.opensearch,
	)
	cnt, ot, err := otelExporter.Get(ctx, iAuth, request.Criterias, request.Paginate)
	if err != nil {
		return exceptions.BadRequestError[protos.GetAllAssistantTelemetryResponse]("Unable to get the assistant telemetry.")
	}
	out := make([]*protos.Telemetry, 0)
	for _, v := range ot {
		out = append(out, v.ToProto())
	}

	return utils.PaginatedSuccess[protos.GetAllAssistantTelemetryResponse, []*protos.Telemetry](
		uint32(cnt),
		request.GetPaginate().GetPage(),
		out)

}

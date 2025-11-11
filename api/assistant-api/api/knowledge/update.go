package knowledge_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

// UpdateKnowledgeDocumentSegment implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) UpdateKnowledgeDocumentSegment(ctx context.Context, dsr *knowledge_api.UpdateKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.BaseResponse](
			errors.New("unauthenticated request for update document segment"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	_, err := knowledgeApi.knowledgeDocumentService.UpdateDocumentSegment(ctx,
		iAuth,
		dsr.GetIndex(),
		dsr.GetDocumentId(),
		dsr.GetDocumentName(),
		dsr.GetOrganizations(),
		dsr.GetDates(),
		dsr.GetProducts(),
		dsr.GetEvents(),
		dsr.GetPeople(),
		dsr.GetTimes(),
		dsr.GetQuantities(),
		dsr.GetLocations(),
		dsr.GetIndustries(),
	)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to delete knowledge segment with error %v", err)
		return utils.Error[knowledge_api.BaseResponse](
			err,
			"Unable to update document segment, please try again.",
		)
	}
	return utils.JustSuccess()
}

// UpdateKnowledgeDetail implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) UpdateKnowledgeDetail(ctx context.Context, cer *knowledge_api.UpdateKnowledgeDetailRequest) (*knowledge_api.GetKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	kn, err := knowledgeApi.knowledgeService.UpdateKnowledgeDetail(ctx, iAuth, cer.GetKnowledgeId(), cer.GetName(), cer.GetDescription())
	if err != nil {
		knowledgeApi.logger.Errorf("unable to update knowledge details with error %v", err)
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			err,
			"Unable to update knowledge details, please try again.",
		)
	}

	_kn, err := knowledgeApi.knowledgeService.Get(ctx, iAuth, kn.Id)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to get knowledge with error %v", err)
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			err,
			"Unable to get knowledge, please try again later.",
		)
	}
	out := &knowledge_api.Knowledge{}
	err = utils.Cast(_kn, out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast the knowledge model to the response object")
	}
	return utils.Success[knowledge_api.GetKnowledgeResponse, *knowledge_api.Knowledge](out)
}

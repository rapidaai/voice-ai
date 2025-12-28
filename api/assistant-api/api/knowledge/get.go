// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	"errors"

	"github.com/rapidaai/pkg/exceptions"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

// GetAllKnowledgeDocument implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeDocument(ctx context.Context, cer *knowledge_api.GetAllKnowledgeDocumentRequest) (*knowledge_api.GetAllKnowledgeDocumentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	cnt, _kns, err := knowledgeApi.knowledgeDocumentService.GetAll(ctx, iAuth, cer.GetKnowledgeId(), cer.GetCriterias(), cer.GetPaginate())
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentResponse](
			err,
			"Unable to get Knowledge Document, please try again later.",
		)
	}

	out := []*knowledge_api.KnowledgeDocument{}
	err = utils.Cast(_kns, &out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast knowledge document %v", err)
	}
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeDocumentResponse](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}

func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeDocumentSegment(ctx context.Context, cer *knowledge_api.GetAllKnowledgeDocumentSegmentRequest) (*knowledge_api.GetAllKnowledgeDocumentSegmentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	knowledge, err := knowledgeApi.knowledgeService.Get(ctx, iAuth, cer.GetKnowledgeId())
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
			err,
			"Unable to get Knowledge, or you do not have access to the knowledge.",
		)
	}

	cnt, _kns, err := knowledgeApi.knowledgeDocumentService.GetAllDocumentSegment(
		ctx,
		iAuth,
		cer.GetKnowledgeId(),
		knowledge.StorageNamespace,
		cer.GetCriterias(),
		cer.GetPaginate(),
	)
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
			err,
			"Unable to get Knowledge Document Segment, please try again later.",
		)
	}
	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeDocumentSegmentResponse](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		_kns)
}

func (knowledgeApi *knowledgeGrpcApi) GetKnowledgeLog(ctx context.Context, cepm *knowledge_api.GetKnowledgeLogRequest) (*knowledge_api.GetKnowledgeLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for GetKnowledgeLogRequest")
		return utils.Error[knowledge_api.GetKnowledgeLogResponse](
			errors.New("unauthenticated request for get assistant converstaion"),
			"Please provider valid service credentials to perform GetKnowledgeLogRequest, read docs @ docs.rapida.ai",
		)
	}
	lg, err := knowledgeApi.knowledgeService.GetLog(
		ctx,
		iAuth,
		cepm.GetProjectId(), cepm.GetId())
	if err != nil {
		return utils.Error[knowledge_api.GetKnowledgeLogResponse](
			err,
			"Unable to get the assistant for given assistant id.",
		)
	}
	wl := &knowledge_api.KnowledgeLog{}
	err = utils.Cast(lg, wl)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast the assistant ToolLog to the response object")
	}

	//

	re, rs, _ := knowledgeApi.knowledgeService.GetLogObject(ctx, *iAuth.GetCurrentOrganizationId(),
		*iAuth.GetCurrentProjectId(), cepm.GetId())
	// if err != nil {
	if re != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(re)
		if err != nil {
			knowledgeApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Request = s
	}
	if rs != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(rs)
		if err != nil {
			knowledgeApi.logger.Errorf("unable to cast the request %v", err)
		}
		wl.Response = s
	}

	return utils.Success[knowledge_api.GetKnowledgeLogResponse, *knowledge_api.KnowledgeLog](wl)

}

func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledgeLog(ctx context.Context, gaar *knowledge_api.GetAllKnowledgeLogRequest) (*knowledge_api.GetAllKnowledgeLogResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return exceptions.AuthenticationError[knowledge_api.GetAllKnowledgeLogResponse]()
	}
	cnt, epms, err := knowledgeApi.knowledgeService.GetAllLog(ctx,
		iAuth,
		gaar.GetProjectId(),
		gaar.GetCriterias(),
		gaar.GetPaginate(),
		gaar.GetOrder())
	if err != nil {
		return exceptions.BadRequestError[knowledge_api.GetAllKnowledgeLogResponse]("Unable to get the assistant for given assistant id.")
	}
	out := []*knowledge_api.KnowledgeLog{}
	err = utils.Cast(epms, &out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast assistant webhook logs %v", err)
	}

	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeLogResponse, []*knowledge_api.KnowledgeLog](
		uint32(cnt),
		gaar.GetPaginate().GetPage(),
		out)
}

// GetKnowledge implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) GetKnowledge(ctx context.Context, cer *knowledge_api.GetKnowledgeRequest) (*knowledge_api.GetKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetKnowledgeResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	_kn, err := knowledgeApi.knowledgeService.Get(ctx, iAuth, cer.GetId())
	if err != nil {
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

// GetAllKnowledge implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) GetAllKnowledge(ctx context.Context, cer *knowledge_api.GetAllKnowledgeRequest) (*knowledge_api.GetAllKnowledgeResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !iAuth.HasProject() {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.GetAllKnowledgeResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}
	cnt, _kns, err := knowledgeApi.knowledgeService.GetAll(ctx, iAuth, cer.GetCriterias(), cer.GetPaginate())
	if err != nil {
		return utils.Error[knowledge_api.GetAllKnowledgeResponse](
			err,
			"Unable to get knowledge, please try again later.",
		)
	}

	out := []*knowledge_api.Knowledge{}
	err = utils.Cast(_kns, &out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast knowledge provider model %v", err)
	}

	for _, kn := range out {
		documentCount, wordCount, tokenCount := knowledgeApi.knowledgeDocumentService.GetCounts(ctx, iAuth, kn.Id)
		kn.DocumentCount = documentCount
		kn.WordCount = wordCount
		kn.TokenCount = tokenCount
	}

	return utils.PaginatedSuccess[knowledge_api.GetAllKnowledgeResponse](
		uint32(cnt),
		cer.GetPaginate().GetPage(),
		out)
}

// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package knowledge_api

import (
	"context"
	"errors"

	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	knowledge_api "github.com/rapidaai/protos"
)

// CreateKnowledgeDocument implements knowledge_api.KnowledgeServiceServer.
func (knowledgeApi *knowledgeGrpcApi) CreateKnowledgeDocument(ctx context.Context, cer *knowledge_api.CreateKnowledgeDocumentRequest) (*knowledge_api.CreateKnowledgeDocumentResponse, error) {
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(ctx)
	if !isAuthenticated || !hasProjectCapability(iAuth) {
		knowledgeApi.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[knowledge_api.CreateKnowledgeDocumentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	kd, err := knowledgeApi.knowledgeService.Get(ctx, iAuth, cer.GetKnowledgeId())
	if err != nil {
		knowledgeApi.logger.Errorf("unable to get knowledge with error %v", err)
		return utils.Error[knowledge_api.CreateKnowledgeDocumentResponse](
			errors.New("unauthenticated request for invoke"),
			"Please provide a valid knowledge to create document.",
		)
	}

	var _kn []*internal_knowledge_gorm.KnowledgeDocument
	switch cer.GetDocumentSource() {
	case knowledge_api.CreateKnowledgeDocumentRequest_DOCUMENT_SOURCE_MANUAL:
		_kn, err = knowledgeApi.knowledgeDocumentService.CreateManualDocument(ctx, iAuth,
			kd,
			cer.GetDataSource(),
			cer.GetDocumentStructure(),
			cer.GetContents(),
		)
		if err != nil {
			knowledgeApi.logger.Errorf("unable to create manual document with error %v", err)
			return utils.Error[knowledge_api.CreateKnowledgeDocumentResponse](
				err,
				"Unable to create Knowledge Document, please try again later.",
			)
		}

		var docIds []uint64
		for _, doc := range _kn {
			docIds = append(docIds, doc.Id)
		}
		knowledgeApi.indexerServiceClient.IndexKnowledgeDocument(ctx, iAuth,
			&knowledge_api.IndexKnowledgeDocumentRequest{
				KnowledgeId:         kd.Id,
				KnowledgeDocumentId: docIds,
			})
	case knowledge_api.CreateKnowledgeDocumentRequest_DOCUMENT_SOURCE_TOOL:
		knowledgeApi.logger.Debugf("calling for create tool document")
		_kn, err = knowledgeApi.knowledgeDocumentService.CreateToolDocument(ctx, iAuth,
			kd,
			cer.GetDataSource(),
			cer.GetDocumentStructure(),
			cer.GetContents(),
		)
		if err != nil {
			knowledgeApi.logger.Errorf("unable to create knowledge with tool document %v", err)
			return utils.Error[knowledge_api.CreateKnowledgeDocumentResponse](
				err,
				"Unable to create Knowledge Document, please try again later.",
			)
		}
	default:
		knowledgeApi.logger.Errorf("unknown datasource for adding file to knowledge %v", err)
		return utils.Error[knowledge_api.CreateKnowledgeDocumentResponse](
			err,
			"Illegal datasource for connecting document.",
		)
	}

	kd.KnowledgeDocuments = _kn
	out := []*knowledge_api.KnowledgeDocument{}
	err = utils.Cast(_kn, &out)
	if err != nil {
		knowledgeApi.logger.Errorf("unable to cast the knowledge model to the response object")
	}
	return utils.Success[knowledge_api.CreateKnowledgeDocumentResponse, []*knowledge_api.KnowledgeDocument](out)
}

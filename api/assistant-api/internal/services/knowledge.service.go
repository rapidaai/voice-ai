// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_services

import (
	"context"

	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

type KnowledgeService interface {
	GetAll(ctx context.Context, auth *types.Authentication, criterias []*protos.Criteria, paginate *protos.Paginate) (int64, *[]internal_knowledge_gorm.Knowledge, error)
	Get(ctx context.Context, auth *types.Authentication, knowledgeId uint64) (*internal_knowledge_gorm.Knowledge, error)
	CreateKnowledge(ctx context.Context, auth *types.Authentication,
		name string, description, visibility *string,
		embeddingProviderModelName string,
		embeddingProviderModelOptions []*protos.Metadata,
	) (*internal_knowledge_gorm.Knowledge, error)
	CreateOrUpdateKnowledgeTag(ctx context.Context,
		auth *types.Authentication,
		knowledgeId uint64,
		tags []string,
	) (*internal_knowledge_gorm.KnowledgeTag, error)
	UpdateKnowledgeDetail(ctx context.Context,
		auth *types.Authentication,
		knowledgeId uint64,
		name, description string) (*internal_knowledge_gorm.Knowledge, error)

	//

	CreateLog(
		ctx context.Context,
		auth *types.Authentication,
		knowledgeId uint64,
		retrievalMethod string,
		topK uint32,
		scoreThreshold float32,
		documentCount int,
		timeTaken int64,
		additionalData map[string]string,
		status type_enums.RecordState,
		request, response []byte,
	) (*internal_knowledge_gorm.KnowledgeLog, error)

	GetLog(
		ctx context.Context,
		auth *types.Authentication,
		projectId uint64,
		knowledgeLogId uint64) (*internal_knowledge_gorm.KnowledgeLog, error)

	GetAllLog(
		ctx context.Context,
		auth *types.Authentication,
		projectId uint64,
		criterias []*protos.Criteria,
		paginate *protos.Paginate,
		order *protos.Ordering) (int64, []*internal_knowledge_gorm.KnowledgeLog, error)

	GetLogObject(
		ctx context.Context,
		auth *types.Authentication,
		toolLogId uint64) (requestData []byte, responseData []byte, err error)
}

type KnowledgeDocumentService interface {
	GetAll(ctx context.Context, auth *types.Authentication,
		knowledgeId uint64,
		criterias []*protos.Criteria, paginate *protos.Paginate) (int64, *[]internal_knowledge_gorm.KnowledgeDocument, error)
	Get(ctx context.Context, auth *types.Authentication, knowledgeId uint64, knowledgeDocumentId uint64) (*internal_knowledge_gorm.KnowledgeDocument, error)
	CreateManualDocument(ctx context.Context,
		auth *types.Authentication,
		knowledge *internal_knowledge_gorm.Knowledge,
		datasource string,
		documentStructure string,
		contents []*protos.DocumentContent) ([]*internal_knowledge_gorm.KnowledgeDocument, error)

	CreateToolDocument(ctx context.Context,
		auth *types.Authentication,
		knowledge *internal_knowledge_gorm.Knowledge,
		datasource string,
		documentStructure string,
		contents []*protos.DocumentContent,
	) ([]*internal_knowledge_gorm.KnowledgeDocument, error)

	GetCounts(ctx context.Context, auth *types.Authentication, knowledgeId uint64) (documentCount, wordCount, tokenCount uint32)
	GetAllDocumentSegment(
		ctx context.Context,
		auth *types.Authentication,
		knowledgeId uint64,
		storageNamespace string,
		criterias []*protos.Criteria,
		paginate *protos.Paginate) (int64, []*protos.KnowledgeDocumentSegment, error)

	UpdateDocumentSegment(
		ctx context.Context,
		auth *types.Authentication,
		index string,
		documentId string,
		documentName string,
		organizations []string,
		dates []string,
		products []string,
		events []string,
		people []string,
		times []string,
		quantities []string,
		locations []string,
		industries []string,
	) (*protos.KnowledgeDocumentSegment, error)

	DeleteDocumentSegment(
		ctx context.Context,
		auth *types.Authentication,
		index string,
		documentId string,
		reason string,
	) (*protos.KnowledgeDocumentSegment, error)
}

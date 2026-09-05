// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package knowledge_api

import (
	"context"
	"errors"
	"testing"

	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	document_client "github.com/rapidaai/pkg/clients/document"
	rapida_client "github.com/rapidaai/pkg/clients/rapida"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	knowledge_api "github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

type knowledgeServiceStub struct {
	internal_services.KnowledgeService
	knowledge *internal_knowledge_gorm.Knowledge
}

func (stub *knowledgeServiceStub) Get(context.Context, *types.Authentication, uint64) (*internal_knowledge_gorm.Knowledge, error) {
	return stub.knowledge, nil
}

type knowledgeDocumentServiceStub struct {
	internal_services.KnowledgeDocumentService
	documents []*internal_knowledge_gorm.KnowledgeDocument
}

func (stub *knowledgeDocumentServiceStub) CreateManualDocument(
	context.Context,
	*types.Authentication,
	*internal_knowledge_gorm.Knowledge,
	string,
	string,
	[]*knowledge_api.DocumentContent,
) ([]*internal_knowledge_gorm.KnowledgeDocument, error) {
	return stub.documents, nil
}

type indexerServiceStub struct {
	document_client.IndexerServiceClient
	request *knowledge_api.IndexKnowledgeDocumentRequest
	err     error
}

func (stub *indexerServiceStub) IndexKnowledgeDocument(
	_ context.Context,
	_ *types.Authentication,
	request *knowledge_api.IndexKnowledgeDocumentRequest,
) (*knowledge_api.IndexKnowledgeDocumentResponse, error) {
	stub.request = request
	return &knowledge_api.IndexKnowledgeDocumentResponse{}, stub.err
}

func TestControllersRetainRapidaClient(t *testing.T) {
	client := &rapida_client.RapidaClient{}

	knowledge := &knowledgeApi{rapidaClient: client}
	if knowledge.rapidaClient != client {
		t.Fatal("knowledge controller did not retain RapidaClient")
	}

	indexer := &indexerApi{rapidaClient: client}
	if indexer.rapidaClient != client {
		t.Fatal("document controller did not retain RapidaClient")
	}
}

func TestCreateKnowledgeDocumentIndexesManualDocuments(t *testing.T) {
	indexer := &indexerServiceStub{}
	api := newKnowledgeDocumentTestAPI(t, indexer)

	response, err := api.CreateKnowledgeDocument(knowledgeTestContext(), &knowledge_api.CreateKnowledgeDocumentRequest{
		KnowledgeId:    10,
		DocumentSource: knowledge_api.CreateKnowledgeDocumentRequest_DOCUMENT_SOURCE_MANUAL,
	})

	require.NoError(t, err)
	require.True(t, response.GetSuccess())
	require.Equal(t, uint64(10), indexer.request.GetKnowledgeId())
	require.Equal(t, []uint64{20}, indexer.request.GetKnowledgeDocumentId())
}

func TestCreateKnowledgeDocumentReturnsIndexerFailure(t *testing.T) {
	indexErr := errors.New("indexing failed")
	indexer := &indexerServiceStub{err: indexErr}
	api := newKnowledgeDocumentTestAPI(t, indexer)

	response, err := api.CreateKnowledgeDocument(knowledgeTestContext(), &knowledge_api.CreateKnowledgeDocumentRequest{
		KnowledgeId:    10,
		DocumentSource: knowledge_api.CreateKnowledgeDocumentRequest_DOCUMENT_SOURCE_MANUAL,
	})

	require.ErrorIs(t, err, indexErr)
	require.False(t, response.GetSuccess())
	require.NotNil(t, indexer.request)
}

func newKnowledgeDocumentTestAPI(t *testing.T, indexer document_client.IndexerServiceClient) *knowledgeGrpcApi {
	t.Helper()
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	require.NoError(t, err)
	knowledge := &internal_knowledge_gorm.Knowledge{}
	knowledge.Id = 10
	document := &internal_knowledge_gorm.KnowledgeDocument{}
	document.Id = 20

	return &knowledgeGrpcApi{knowledgeApi: knowledgeApi{
		logger:                   logger,
		knowledgeService:         &knowledgeServiceStub{knowledge: knowledge},
		knowledgeDocumentService: &knowledgeDocumentServiceStub{documents: []*internal_knowledge_gorm.KnowledgeDocument{document}},
		rapidaClient:             &rapida_client.RapidaClient{Indexer: indexer},
	}}
}

func knowledgeTestContext() context.Context {
	organizationID := uint64(1)
	projectID := uint64(2)
	return context.WithValue(context.Background(), types.CTX_, &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: projectID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	})
}

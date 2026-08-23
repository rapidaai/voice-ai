// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package workflow_client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	clients "github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	knowledge_api "github.com/rapidaai/protos"
)

type KnowledgeServiceClient interface {
	GetAllKnowledge(c context.Context, auth *types.Authentication, criteria []*knowledge_api.Criteria, paginate *knowledge_api.Paginate) (*knowledge_api.Paginated, []*knowledge_api.Knowledge, error)
	GetKnowledge(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.GetKnowledgeRequest) (*knowledge_api.Knowledge, error)
	CreateKnowledge(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.CreateKnowledgeRequest) (*knowledge_api.CreateKnowledgeResponse, error)
	CreateKnowledgeTag(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.CreateKnowledgeTagRequest) (*knowledge_api.GetKnowledgeResponse, error)
	UpdateKnowledgeDetail(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.UpdateKnowledgeDetailRequest) (*knowledge_api.GetKnowledgeResponse, error)

	CreateKnowledgeDocument(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.CreateKnowledgeDocumentRequest) (*knowledge_api.CreateKnowledgeDocumentResponse, error)
	GetAllKnowledgeDocument(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.GetAllKnowledgeDocumentRequest) (*knowledge_api.GetAllKnowledgeDocumentResponse, error)
	GetAllKnowledgeDocumentSegment(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.GetAllKnowledgeDocumentSegmentRequest) (*knowledge_api.GetAllKnowledgeDocumentSegmentResponse, error)

	UpdateKnowledgeDocumentSegment(ctx context.Context, auth *types.Authentication, dsr *knowledge_api.UpdateKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error)
	DeleteKnowledgeDocumentSegment(ctx context.Context, auth *types.Authentication, dsr *knowledge_api.DeleteKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error)

	GetAllKnowledgeLog(ctx context.Context, auth *types.Authentication, in *knowledge_api.GetAllKnowledgeLogRequest) (*knowledge_api.GetAllKnowledgeLogResponse, error)
	GetKnowledgeLog(ctx context.Context, auth *types.Authentication, in *knowledge_api.GetKnowledgeLogRequest) (*knowledge_api.GetKnowledgeLogResponse, error)
}

type knowledgeServiceClient struct {
	clients.InternalClient
	cfg             *config.AppConfig
	logger          commons.Logger
	knowledgeClient knowledge_api.KnowledgeServiceClient
}

func NewKnowledgeServiceClientGRPC(config *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) KnowledgeServiceClient {
	grpcOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(commons.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(commons.MaxSendMsgSize),
		),
	}
	conn, err := grpc.NewClient(config.Assistant.Host,
		grpcOpts...)

	if err != nil {
		logger.Fatalf("Unable to create connection %v", err)
	}
	return &knowledgeServiceClient{
		InternalClient:  clients.NewInternalClient(config, logger, redis),
		cfg:             config,
		logger:          logger,
		knowledgeClient: knowledge_api.NewKnowledgeServiceClient(conn),
	}
}

func (client *knowledgeServiceClient) GetAllKnowledge(c context.Context, auth *types.Authentication, criteria []*knowledge_api.Criteria, paginate *knowledge_api.Paginate) (*knowledge_api.Paginated, []*knowledge_api.Knowledge, error) {
	client.logger.Debugf("get all knowledge request")
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, nil, err
	}
	res, err := client.knowledgeClient.GetAllKnowledge(authContext, &knowledge_api.GetAllKnowledgeRequest{
		Paginate:  paginate,
		Criterias: criteria,
	})
	if err != nil {
		client.logger.Errorf("error while calling to get all knowledge %v", err)
		return nil, nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get all knowledge %v", err)
		return nil, nil, err
	}

	return res.GetPaginated(), res.GetData(), nil
}

func (client *knowledgeServiceClient) GetKnowledge(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.GetKnowledgeRequest) (*knowledge_api.Knowledge, error) {
	client.logger.Debugf("get knowledge request")
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.GetKnowledge(authContext, knowledgeRequest)
	if err != nil {
		client.logger.Errorf("error while calling to get knowledge %v", err)
		return nil, err
	}
	if !res.GetSuccess() {
		client.logger.Errorf("error while calling to get knowledge %v", err)
		return nil, err
	}

	return res.GetData(), nil
}

func (client *knowledgeServiceClient) CreateKnowledge(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.CreateKnowledgeRequest) (*knowledge_api.CreateKnowledgeResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.CreateKnowledge(authContext, knowledgeRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateKnowledge %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) CreateKnowledgeTag(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.CreateKnowledgeTagRequest) (*knowledge_api.GetKnowledgeResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.CreateKnowledgeTag(authContext, knowledgeRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateKnowledgeTag %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) UpdateKnowledgeDetail(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.UpdateKnowledgeDetailRequest) (*knowledge_api.GetKnowledgeResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.UpdateKnowledgeDetail(authContext, knowledgeRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateKnowledgeTag %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) CreateKnowledgeDocument(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.CreateKnowledgeDocumentRequest) (*knowledge_api.CreateKnowledgeDocumentResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.CreateKnowledgeDocument(authContext, knowledgeRequest)
	if err != nil {
		client.logger.Errorf("error while calling CreateKnowledgeDocument %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) GetAllKnowledgeDocument(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.GetAllKnowledgeDocumentRequest) (*knowledge_api.GetAllKnowledgeDocumentResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.GetAllKnowledgeDocument(authContext, knowledgeRequest)
	if err != nil {
		client.logger.Errorf("error while calling GetAllKnowledgeDocument %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) GetAllKnowledgeDocumentSegment(c context.Context, auth *types.Authentication, knowledgeRequest *knowledge_api.GetAllKnowledgeDocumentSegmentRequest) (*knowledge_api.GetAllKnowledgeDocumentSegmentResponse, error) {
	authContext, err := client.WithAuth(c, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.GetAllKnowledgeDocumentSegment(authContext, knowledgeRequest)
	if err != nil {
		client.logger.Errorf("error while calling GetAllKnowledgeDocumentSegment %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) UpdateKnowledgeDocumentSegment(ctx context.Context, auth *types.Authentication, dsr *knowledge_api.UpdateKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error) {
	authContext, err := client.WithAuth(ctx, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.UpdateKnowledgeDocumentSegment(authContext, dsr)
	if err != nil {
		client.logger.Errorf("error while calling GetAllKnowledgeDocumentSegment %v", err)
		return nil, err
	}
	return res, nil
}
func (client *knowledgeServiceClient) DeleteKnowledgeDocumentSegment(ctx context.Context, auth *types.Authentication, dsr *knowledge_api.DeleteKnowledgeDocumentSegmentRequest) (*knowledge_api.BaseResponse, error) {
	authContext, err := client.WithAuth(ctx, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.DeleteKnowledgeDocumentSegment(authContext, dsr)
	if err != nil {
		client.logger.Errorf("error while calling GetAllKnowledgeDocumentSegment %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) GetAllKnowledgeLog(ctx context.Context, auth *types.Authentication, in *knowledge_api.GetAllKnowledgeLogRequest) (*knowledge_api.GetAllKnowledgeLogResponse, error) {
	authContext, err := client.WithAuth(ctx, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.GetAllKnowledgeLog(authContext, in)
	if err != nil {
		client.logger.Errorf("error while calling GetAllKnowledgeLog %v", err)
		return nil, err
	}
	return res, nil
}

func (client *knowledgeServiceClient) GetKnowledgeLog(ctx context.Context, auth *types.Authentication, in *knowledge_api.GetKnowledgeLogRequest) (*knowledge_api.GetKnowledgeLogResponse, error) {
	authContext, err := client.WithAuth(ctx, auth)
	if err != nil {
		return nil, err
	}
	res, err := client.knowledgeClient.GetKnowledgeLog(authContext, in)
	if err != nil {
		client.logger.Errorf("error while calling GetKnowledgeLog %v", err)
		return nil, err
	}
	return res, nil
}

package web_api

import (
	"context"
	"errors"

	internal_productusage_service "github.com/rapidaai/api/web-api/internal/service/productusage"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type webProductUsageGRPCApi struct {
	protos.UnimplementedProductUsageServiceServer
	logger              commons.Logger
	productUsageService internal_productusage_service.Service
}

func NewProductUsageGRPC(logger commons.Logger, postgres connectors.PostgresConnector) protos.ProductUsageServiceServer {
	return &webProductUsageGRPCApi{
		logger:              logger,
		productUsageService: internal_productusage_service.NewProductUsageService(postgres),
	}
}

func (api *webProductUsageGRPCApi) CreateProductUsages(ctx context.Context, request *protos.CreateProductUsagesRequest) (*protos.CreateProductUsagesResponse, error) {
	auth, err := types.Authorize(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	auth, err = auth.Scope(types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	if _, err = auth.ProjectContext(); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	result, err := api.productUsageService.CreateProductUsages(ctx, auth, request.GetUsages())
	if err != nil {
		switch {
		case errors.Is(err, internal_productusage_service.ErrInvalidRequest):
			return nil, status.Error(codes.InvalidArgument, err.Error())
		case errors.Is(err, internal_productusage_service.ErrProjectMismatch):
			return nil, status.Error(codes.PermissionDenied, err.Error())
		case errors.Is(err, internal_productusage_service.ErrUsageConflict):
			return nil, status.Error(codes.AlreadyExists, err.Error())
		default:
			api.logger.Errorf("unable to create product usages: %v", err)
			return nil, status.Error(codes.Internal, "unable to create product usages")
		}
	}

	return &protos.CreateProductUsagesResponse{
		Code:           200,
		Success:        true,
		CreatedCount:   result.CreatedCount,
		DuplicateCount: result.DuplicateCount,
	}, nil
}

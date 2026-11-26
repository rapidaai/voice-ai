package web_api

import (
	"context"
	"errors"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_productusage_service "github.com/rapidaai/api/web-api/internal/service/productusage"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	middleware_authentication "github.com/rapidaai/pkg/middlewares/authentication"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type webProductUsageGRPCApi struct {
	protos.UnimplementedProductUsageServiceServer
	logger              commons.Logger
	productUsageService internal_productusage_service.Service
}

func NewProductUsageGRPC(logger commons.Logger, postgres connectors.PostgresConnector) protos.ProductUsageServiceServer {
	return &webProductUsageGRPCApi{
		logger:              logger,
		productUsageService: internal_productusage_service.NewProductUsageService(postgres, logger),
	}
}

func (api *webProductUsageGRPCApi) CreateProductUsage(ctx context.Context, request *protos.CreateProductUsageRequest) (*protos.GetProductUsageResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if principal.Tenant().ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}

	usage, err := api.productUsageService.CreateProductUsage(ctx, principal, request)
	if err != nil {
		return nil, api.productUsageError("create", err)
	}
	return &protos.GetProductUsageResponse{Code: 200, Success: true, Data: productUsageProto(usage)}, nil
}

func (api *webProductUsageGRPCApi) GetProductUsages(ctx context.Context, request *protos.GetProductUsagesRequest) (*protos.GetUsagesResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if principal.Tenant().ProjectID == 0 {
		return nil, status.Error(codes.PermissionDenied, "project context is required")
	}

	count, usages, err := api.productUsageService.GetProductUsages(ctx, principal, request.GetUsageType(), request.GetCriterias(), request.GetPaginate())
	if err != nil {
		return nil, api.productUsageError("get project", err)
	}
	return productUsagesResponse(count, request.GetPaginate(), usages)
}

func (api *webProductUsageGRPCApi) GetOrganizationUsages(ctx context.Context, request *protos.GetOrganizationUsagesRequest) (*protos.GetUsagesResponse, error) {
	principal, ok := middleware_authentication.Principal(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "valid authentication is required")
	}
	if principal.Tenant().OrganizationID == 0 {
		return nil, status.Error(codes.PermissionDenied, "organization context is required")
	}

	count, usages, err := api.productUsageService.GetOrganizationUsages(ctx, principal, request.GetCriterias(), request.GetPaginate())
	if err != nil {
		return nil, api.productUsageError("get organization", err)
	}
	return productUsagesResponse(count, request.GetPaginate(), usages)
}

func (api *webProductUsageGRPCApi) productUsageError(operation string, err error) error {
	switch {
	case errors.Is(err, internal_productusage_service.ErrInvalidRequest):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, internal_productusage_service.ErrProjectMismatch):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		api.logger.Errorf("unable to %s product usage: %v", operation, err)
		return status.Errorf(codes.Internal, "unable to %s product usage", operation)
	}
}

func productUsagesResponse(count int64, paginate *protos.Paginate, usages []*internal_entity.ProductUsage) (*protos.GetUsagesResponse, error) {
	totalItem, err := utils.Int64ToUint32(count)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid product usage count")
	}

	data := make([]*protos.ProductUsage, len(usages))
	for index, usage := range usages {
		data[index] = productUsageProto(usage)
	}
	return &protos.GetUsagesResponse{
		Code:    200,
		Success: true,
		Data:    data,
		Paginated: &protos.Paginated{
			CurrentPage: paginate.GetPage(),
			TotalItem:   totalItem,
		},
	}, nil
}

func productUsageProto(usage *internal_entity.ProductUsage) *protos.ProductUsage {
	if usage == nil {
		return nil
	}
	return &protos.ProductUsage{
		Id:         usage.Id,
		ProjectId:  usage.ProjectId,
		UsageType:  usage.UsageType,
		Usages:     usage.Usages,
		Unit:       usage.Unit,
		OccurredAt: timestamppb.New(usage.OccurredAt.UTC()),
	}
}

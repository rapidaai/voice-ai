package web_api

import (
	"context"
	"errors"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_productusage_service "github.com/rapidaai/api/web-api/internal/service/productusage"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
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
		productUsageService: internal_productusage_service.NewProductUsageService(postgres),
	}
}

func (api *webProductUsageGRPCApi) CreateProductUsage(ctx context.Context, request *protos.CreateProductUsageRequest) (*protos.GetProductUsageResponse, error) {
	auth, err := authorizeProductUsage(ctx, types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if err != nil {
		return nil, err
	}
	if _, err = auth.ProjectContext(); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	usage, err := api.productUsageService.CreateProductUsage(ctx, auth, request)
	if err != nil {
		return nil, api.productUsageError("create", err)
	}
	return &protos.GetProductUsageResponse{Code: 200, Success: true, Data: productUsageProto(usage)}, nil
}

func (api *webProductUsageGRPCApi) GetProductUsages(ctx context.Context, request *protos.GetProductUsagesRequest) (*protos.GetUsagesResponse, error) {
	auth, err := authorizeProductUsage(ctx, types.AuthTypeUser, types.AuthTypeProject, types.AuthTypeService)
	if err != nil {
		return nil, err
	}
	if _, err = auth.ProjectContext(); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	count, usages, err := api.productUsageService.GetProductUsages(ctx, auth, request.GetUsageType(), request.GetCriterias(), request.GetPaginate())
	if err != nil {
		return nil, api.productUsageError("get project", err)
	}
	return productUsagesResponse(count, request.GetPaginate(), usages)
}

func (api *webProductUsageGRPCApi) GetOrganizationUsages(ctx context.Context, request *protos.GetOrganizationUsagesRequest) (*protos.GetUsagesResponse, error) {
	auth, err := authorizeProductUsage(ctx, types.AuthTypeUser, types.AuthTypeOrg, types.AuthTypeService)
	if err != nil {
		return nil, err
	}
	if _, err = auth.OrganizationContext(); err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	count, usages, err := api.productUsageService.GetOrganizationUsages(ctx, auth, request.GetCriterias(), request.GetPaginate())
	if err != nil {
		return nil, api.productUsageError("get organization", err)
	}
	return productUsagesResponse(count, request.GetPaginate(), usages)
}

func authorizeProductUsage(ctx context.Context, allowed ...types.AuthType) (*types.Authentication, error) {
	auth, err := types.Authorize(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}
	auth, err = auth.Scope(allowed...)
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}
	return auth, nil
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

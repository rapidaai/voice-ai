package integration_api

import (
	"context"
	"errors"

	config "github.com/rapidaai/api/integration-api/config"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
	"google.golang.org/protobuf/types/known/structpb"
)

type auditLoggingApi struct {
	integrationApi
}

type auditLoggingGRPCApi struct {
	auditLoggingApi
}

func NewAuditLoggingGRPC(config *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integration_api.AuditLoggingServiceServer {
	return &auditLoggingGRPCApi{
		auditLoggingApi{
			integrationApi: NewInegrationApi(config, logger, postgres),
		},
	}
}

func (als *auditLoggingGRPCApi) GetAuditLog(c context.Context, ir *integration_api.GetAuditLogRequest) (*integration_api.GetAuditLogResponse, error) {
	als.logger.Debugf("GetAuditLog %+v", ir)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		als.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[integration_api.GetAuditLogResponse](
			errors.New("unauthenticated request for getAuditLog"),
			"Please provider a valid credentials.",
		)
	}

	lg, err := als.auditService.Get(c, *iAuth.GetCurrentOrganizationId(),
		*iAuth.GetCurrentProjectId(), ir.GetId())
	if err != nil {
		als.logger.Errorf("unable to get audit log %v", err)
		return &integration_api.GetAuditLogResponse{
			Code:    200,
			Success: true,
		}, nil
	}

	out := &integration_api.AuditLog{}
	err = utils.Cast(lg, out)
	if err != nil {
		als.logger.Errorf("unable to cast the information to generic struct %v", err)
	}

	re, rs, _ := als.GetRequestAndResponse(c, *iAuth.GetCurrentOrganizationId(),
		*iAuth.GetCurrentProjectId(), lg.CredentialId, lg.Id)
	// if err != nil {
	if re != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(re)
		if err != nil {
			als.logger.Errorf("unable to cast the request %v", err)
		}
		out.Request = s
	}
	if rs != nil {
		s := &structpb.Struct{}
		err = s.UnmarshalJSON(rs)
		if err != nil {
			als.logger.Errorf("unable to cast the request %v", err)
		}
		out.Response = s
	}

	return utils.Success[integration_api.GetAuditLogResponse, *integration_api.AuditLog](out)
}

func (als *auditLoggingGRPCApi) GetAllAuditLog(c context.Context, ir *integration_api.GetAllAuditLogRequest) (*integration_api.GetAllAuditLogResponse, error) {
	als.logger.Debugf("GetAuditLog %+v", ir)
	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		als.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[integration_api.GetAllAuditLogResponse](
			errors.New("unauthenticated request for getAuditLog"),
			"Please provider valid service credentials to perfom invoke, read docs @ docs.rapida.ai",
		)
	}

	cnt, lgs, err := als.auditService.GetAll(c,
		*iAuth.GetCurrentOrganizationId(),
		*iAuth.GetCurrentProjectId(), ir.GetPaginate(), ir.GetCriterias())
	if err != nil {
		return &integration_api.GetAllAuditLogResponse{
			Code:    200,
			Success: true,
		}, nil
	}

	out := make([]*integration_api.AuditLog, 0)
	err = utils.Cast(lgs, &out)
	if err != nil {
		als.logger.Errorf("unable to cast the information to generic struct %v", err)
	}

	return utils.PaginatedSuccess[integration_api.GetAllAuditLogResponse](
		uint32(cnt),
		ir.GetPaginate().GetPage(),
		out)

}

// CreateMetadata implements lexatic_backend.AuditLoggingServiceServer.
func (als *auditLoggingGRPCApi) CreateMetadata(c context.Context, cmr *integration_api.CreateMetadataRequest) (*integration_api.CreateMetadataResponse, error) {

	iAuth, isAuthenticated := types.GetSimplePrincipleGRPC(c)
	if !isAuthenticated || !iAuth.HasProject() {
		als.logger.Errorf("unauthenticated request for invoke")
		return utils.Error[integration_api.CreateMetadataResponse](
			errors.New("unauthenticated request for CreateMetadata"),
			"Please provider valid service credentials to perfom probe, read docs @ docs.rapida.ai",
		)
	}
	_adt, err := als.auditService.Get(c, *iAuth.GetCurrentOrganizationId(), *iAuth.GetCurrentProjectId(), cmr.GetId())
	if err != nil {
		return utils.Error[integration_api.CreateMetadataResponse](
			errors.New("unauthenticated request for CreateMetadata"),
			"Please provider valid service credentials to perfom probe, read docs @ docs.rapida.ai",
		)
	}

	_, err = als.auditService.UpdateMetadata(c, _adt.Id, cmr.GetAdditionalData())
	if err != nil {
		return utils.Error[integration_api.CreateMetadataResponse](
			errors.New("illegal request for update metadata"),
			"Unable to update the metadata, please try again in sometime.",
		)
	}

	out := &integration_api.AuditLog{}
	err = utils.Cast(_adt, out)
	if err != nil {
		als.logger.Errorf("unable to cast the information to generic struct %v", err)
	}

	return utils.Success[integration_api.CreateMetadataResponse, *integration_api.AuditLog](out)
}

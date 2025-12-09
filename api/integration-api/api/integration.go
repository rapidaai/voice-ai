// Rapida – Open Source Voice AI Orchestration Platform
// Copyright (C) 2023-2025 Prashant Srivastav <prashant@rapida.ai>
// Licensed under a modified GPL-2.0. See the LICENSE file for details.
package integration_api

import (
	"context"
	"fmt"
	"time"

	config "github.com/rapidaai/api/integration-api/config"
	internal_services "github.com/rapidaai/api/integration-api/internal/service"
	internal_audit_service "github.com/rapidaai/api/integration-api/internal/service/audit"
	commons "github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/pkg/utils"
	integration_api "github.com/rapidaai/protos"
)

type integrationApi struct {
	cfg          *config.IntegrationConfig
	logger       commons.Logger
	storage      storages.Storage
	auditService internal_services.AuditService
}

func NewInegrationApi(cfg *config.IntegrationConfig, logger commons.Logger, postgres connectors.PostgresConnector) integrationApi {
	return integrationApi{cfg: cfg, logger: logger,
		storage:      storage_files.NewStorage(cfg.AssetStoreConfig, logger),
		auditService: internal_audit_service.NewAuditService(logger, postgres)}
}

func (iApi *integrationApi) ObjectPrefix(orgId, projectId, credentialId uint64) string {
	return fmt.Sprintf("%d/%d/%d", orgId, projectId, credentialId)
}

func (iApi *integrationApi) ObjectKey(keyPrefix string, auditId uint64, objName string) string {
	return fmt.Sprintf("%s/%d__%s", keyPrefix, auditId, objName)
}

type ProviderModelRequest interface {
	GetAdditionalData() map[string]string
	GetCredential() *integration_api.Credential
}

func (iApi *integrationApi) RequestId() uint64 {
	return gorm_generator.ID()
}
func (iApi *integrationApi) PreHook(c context.Context, auth types.SimplePrinciple, irRequest ProviderModelRequest, requestId uint64, intName string) func(rst map[string]interface{}) {
	return func(rst map[string]interface{}) {
		iApi.preHook(c,
			auth,
			irRequest.GetAdditionalData(),
			irRequest.GetCredential().GetId(),
			requestId,
			intName,
			rst)
	}
}

func (iApi *integrationApi) PostHook(c context.Context, auth types.SimplePrinciple, irRequest ProviderModelRequest, requestId uint64, intName string) func(rst map[string]interface{}, metrics types.Metrics) {
	return func(rst map[string]interface{}, metrics types.Metrics) {
		iApi.postHook(c, auth,
			irRequest.GetAdditionalData(),
			irRequest.GetCredential().GetId(),
			requestId,
			intName,
			rst,
			metrics)
	}
}

func (iApi *integrationApi) preHook(ctx context.Context, auth types.SimplePrinciple, extras map[string]string, credentialId, requestId uint64, intName string, request map[string]interface{}) {
	keyPrefix := iApi.ObjectPrefix(*auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), credentialId)
	start := time.Now()
	go func(currentCtx context.Context, requestId uint64, s3Prefix string, additionalData map[string]string, _request map[string]interface{}) {

		nCtx := context.Background()
		_, err := iApi.auditService.Create(nCtx, requestId, *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), credentialId, intName, keyPrefix, []*types.Metric{}, "active")
		if err != nil {
			iApi.logger.Benchmark("integration.PreHook.Execute", time.Since(start))
			iApi.logger.Debugf("Executing prehook for auditId %d with error %+v", requestId, err)
			return
		}
		iApi.logger.Debugf("Executing prehook for auditId %d", requestId)
		_, err = iApi.auditService.CreateMetadata(nCtx, requestId, additionalData)
		if err != nil || _request == nil {
			iApi.logger.Benchmark("integration.PreHook.Execute", time.Since(start))
			iApi.logger.Debugf("Executing prehook for auditId %d with error %+v", requestId, err)
			iApi.logger.Errorf("unable to update metadata err %v", err)
		}
		key := iApi.ObjectKey(s3Prefix, requestId, "request.json")
		_str, err := utils.Serialize(_request)
		if err != nil {
			iApi.logger.Benchmark("integration.PreHook.Execute", time.Since(start))
			iApi.logger.Errorf("unable to create json err %v", err)
			iApi.storage.Store(nCtx, key, []byte("{}"))
			return
		}
		iApi.storage.Store(nCtx, key, _str)

	}(ctx, requestId, keyPrefix, extras, request)
}
func (iApi *integrationApi) postHook(ctx context.Context, auth types.SimplePrinciple, extras map[string]string, credentialId, requestId uint64, intName string, response map[string]interface{}, metrics []*types.Metric) {
	iApi.logger.Debugf("Executing posthook for auditId %d", requestId)
	start := time.Now()
	keyPrefix := iApi.ObjectPrefix(*auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), credentialId)
	go func(currentContext context.Context, _auditId uint64, s3Prefix string, _response map[string]interface{}) {
		<-currentContext.Done()
		nCtx := context.Background()

		_, err := iApi.auditService.UpdateMetadata(nCtx, requestId, extras)
		if err != nil {
			iApi.logger.Errorf("unable to update the external audit metadata table for audit Id %d", _auditId)
			iApi.logger.Benchmark("integration.PostHook.Execute", time.Since(start))

			// need to change with valid random id
		}
		iApi.logger.Debugf("Executing posthook for auditId %d", _auditId)
		_, err = iApi.auditService.Create(nCtx, _auditId, *auth.GetCurrentOrganizationId(), *auth.GetCurrentProjectId(), credentialId, intName, s3Prefix, metrics, "complete")
		if err != nil {
			iApi.logger.Debugf("Executing posthook for auditId %d with error %+v", requestId, err)
			iApi.logger.Errorf("unable to update the external audit table for audit Id %d", _auditId)
			iApi.logger.Benchmark("integration.PostHook.Execute", time.Since(start))
			// need to change with valid random id
		}
		key := iApi.ObjectKey(s3Prefix, _auditId, "response.json")
		if _response == nil {
			iApi.logger.Errorf("response object is null for audit Id %d", _auditId)
			iApi.storage.Store(nCtx, key, []byte("{}"))
			iApi.logger.Benchmark("integration.PostHook.Execute", time.Since(start))
			return
		}

		_str, err := utils.Serialize(_response)
		if err != nil {
			iApi.logger.Errorf("unable to create json err %v", err)
			iApi.storage.Store(nCtx, key, []byte("{}"))
			iApi.logger.Benchmark("integration.PostHook.Execute", time.Since(start))
			return
		}
		iApi.storage.Store(nCtx, key, _str)
	}(ctx, requestId, keyPrefix, response)
}

func (iApi *integrationApi) GetRequestAndResponse(ctx context.Context, organizationId, projectId, credentialId, auditId uint64) (requestData []byte, responseData []byte, err error) {
	keyPrefix := iApi.ObjectPrefix(organizationId, projectId, credentialId)
	responseKey := iApi.ObjectKey(keyPrefix, auditId, "response.json")
	requestKey := iApi.ObjectKey(keyPrefix, auditId, "request.json")

	type _fileStruct struct {
		Key   string
		Data  []byte
		Error error
	}

	responseChan := make(chan _fileStruct)
	requestChan := make(chan _fileStruct)

	go func(key string) {
		iApi.logger.Debugf("Getting key from s3 %s", key)
		result := iApi.storage.Get(ctx, key)
		if result.Error != nil {
			iApi.logger.Errorf("error downloading goroutine: %v", result.Error)
			responseChan <- _fileStruct{Key: key, Error: result.Error}
			close(responseChan)
			return
		}
		responseChan <- _fileStruct{Key: key, Data: result.Data}
		close(responseChan)
	}(responseKey)

	go func(key string) {
		iApi.logger.Debugf("Getting key from s3 %s", key)
		result := iApi.storage.Get(ctx, key)
		if result.Error != nil {
			iApi.logger.Errorf("error downloading goroutine: %v", result.Error)
			requestChan <- _fileStruct{Key: key, Error: result.Error}
			close(requestChan)
			return
		}
		requestChan <- _fileStruct{Key: key, Data: result.Data}
		close(requestChan)

	}(requestKey)

	// wg.Wait()
	// close(requestChan)

	// Read results from the channels
	for result := range responseChan {
		if result.Error != nil {
			iApi.logger.Errorf("error downloading/parsing response: %v", result.Error)
			break
		}
		responseData = result.Data
	}

	for result := range requestChan {
		if result.Error != nil {
			iApi.logger.Errorf("error downloading/parsing request: %v", result.Error)
			break
		}
		requestData = result.Data
	}

	return requestData, responseData, nil
}

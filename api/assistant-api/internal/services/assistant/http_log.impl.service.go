// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_assistant_service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/clients/rest"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_generator "github.com/rapidaai/pkg/models/gorm/generators"
	"github.com/rapidaai/pkg/storages"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type assistantHTTPLogService struct {
	logger   commons.Logger
	postgres connectors.PostgresConnector
	storage  storages.Storage
}

type retryRequestSnapshot struct {
	URL       string                 `json:"url"`
	Method    string                 `json:"method"`
	Headers   map[string]string      `json:"headers"`
	TimeoutMs uint32                 `json:"timeout_ms"`
	Body      map[string]interface{} `json:"body"`
}

func NewAssistantHTTPLogService(
	logger commons.Logger,
	postgres connectors.PostgresConnector,
	storage storages.Storage,
) internal_services.AssistantHTTPLogService {
	return &assistantHTTPLogService{
		logger:   logger,
		postgres: postgres,
		storage:  storage,
	}
}

func (s *assistantHTTPLogService) CreateLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	source string,
	sourceRefId uint64,
	sourceEvent string,
	contextID string,
	assistantId uint64,
	assistantConversationId *uint64,
	httpUrl string,
	httpMethod string,
	responseStatus int64,
	timeTaken int64,
	retryCount uint32,
	status type_enums.RecordState,
	errorMessage *string,
	request []byte,
	response []byte,
) (*internal_assistant_entity.AssistantHTTPLog, error) {
	projectContext, err := types.RequireProject(auth)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	db := s.postgres.DB(ctx)
	assetPrefix := s.ObjectPrefix(projectContext.OrganizationID, projectContext.ProjectID)
	logID := gorm_generator.ID()

	utils.Go(ctx, func() {
		key := s.ObjectKey(assetPrefix, logID, "request.json")
		s.storage.Store(ctx, key, request)
	})

	utils.Go(ctx, func() {
		key := s.ObjectKey(assetPrefix, logID, "response.json")
		s.storage.Store(ctx, key, response)
	})

	httpLog := &internal_assistant_entity.AssistantHTTPLog{
		Audited: gorm_models.Audited{
			Id: logID,
		},
		Source:                  source,
		SourceRefId:             sourceRefId,
		SourceEvent:             sourceEvent,
		ContextId:               contextID,
		AssistantId:             assistantId,
		AssistantConversationId: assistantConversationId,
		HttpMethod:              httpMethod,
		HttpUrl:                 httpUrl,
		AssetPrefix:             assetPrefix,
		ResponseStatus:          responseStatus,
		TimeTaken:               timeTaken,
		RetryCount:              retryCount,
		ErrorMessage:            errorMessage,
		Organizational: gorm_models.Organizational{
			ProjectId:      projectContext.ProjectID,
			OrganizationId: projectContext.OrganizationID,
		},
		Mutable: gorm_models.Mutable{
			Status: status,
		},
	}

	tx := db.Create(httpLog)
	if tx.Error != nil {
		s.logger.Benchmark("assistantHTTPLogService.CreateLog", time.Since(start))
		s.logger.Errorf("error while creating http log %v", tx.Error)
		return nil, tx.Error
	}

	s.logger.Benchmark("assistantHTTPLogService.CreateLog", time.Since(start))
	return httpLog, nil
}

func (s *assistantHTTPLogService) GetLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	httpLogId uint64,
) (*internal_assistant_entity.AssistantHTTPLog, error) {
	projectContext, err := types.RequireProject(auth)
	if err != nil {
		return nil, err
	}
	if projectId != projectContext.ProjectID {
		return nil, fmt.Errorf("project %d does not match authenticated project %d", projectId, projectContext.ProjectID)
	}

	start := time.Now()
	db := s.postgres.DB(ctx)
	var httpLog *internal_assistant_entity.AssistantHTTPLog
	tx := db.Where("id = ? AND organization_id = ? AND project_id = ?", httpLogId, projectContext.OrganizationID, projectContext.ProjectID).
		First(&httpLog)
	if tx.Error != nil {
		s.logger.Benchmark("assistantHTTPLogService.GetLog", time.Since(start))
		s.logger.Errorf("not able to find any http log %v", tx.Error)
		return nil, tx.Error
	}

	s.logger.Benchmark("assistantHTTPLogService.GetLog", time.Since(start))
	return httpLog, nil
}

func (s *assistantHTTPLogService) GetAllLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	criterias []*protos.Criteria,
	paginate *protos.Paginate,
	order *protos.Ordering,
) (int64, []*internal_assistant_entity.AssistantHTTPLog, error) {
	projectContext, err := types.RequireProject(auth)
	if err != nil {
		return 0, nil, err
	}
	if projectId != projectContext.ProjectID {
		return 0, nil, fmt.Errorf("project %d does not match authenticated project %d", projectId, projectContext.ProjectID)
	}

	start := time.Now()
	db := s.postgres.DB(ctx)
	var (
		httpLogs []*internal_assistant_entity.AssistantHTTPLog
		cnt      int64
	)
	qry := db.Model(internal_assistant_entity.AssistantHTTPLog{})
	qry = qry.Where("organization_id = ? AND project_id = ? ", projectContext.OrganizationID, projectContext.ProjectID)
	for _, ct := range criterias {
		if ct == nil || ct.GetKey() == "" || ct.GetValue() == "" {
			continue
		}

		switch ct.GetKey() {
		case "id":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.id = ?", ct.GetValue())
			}
		case "created_date":
			switch ct.GetLogic() {
			case ">=":
				qry = qry.Where("assistant_http_logs.created_date >= ?", ct.GetValue())
			case "<=":
				qry = qry.Where("assistant_http_logs.created_date <= ?", ct.GetValue())
			case "=":
				qry = qry.Where("assistant_http_logs.created_date = ?", ct.GetValue())
			}
		case "source_ref_id":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.source_ref_id = ?", ct.GetValue())
			}
		case "assistant_conversation_id":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.assistant_conversation_id = ?", ct.GetValue())
			}
		case "assistant_id":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.assistant_id = ?", ct.GetValue())
			}
		case "source_event":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.source_event = ?", ct.GetValue())
			case "contains":
				qry = qry.Where("assistant_http_logs.source_event ILIKE ?", "%"+ct.GetValue()+"%")
			}
		case "http_method":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.http_method = ?", ct.GetValue())
			}
		case "http_url":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.http_url = ?", ct.GetValue())
			case "contains":
				qry = qry.Where("assistant_http_logs.http_url ILIKE ?", "%"+ct.GetValue()+"%")
			}
		case "response_status":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.response_status = ?", ct.GetValue())
			case ">=":
				qry = qry.Where("assistant_http_logs.response_status >= ?", ct.GetValue())
			case "<=":
				qry = qry.Where("assistant_http_logs.response_status <= ?", ct.GetValue())
			}
		case "time_taken":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.time_taken = ?", ct.GetValue())
			case ">=":
				qry = qry.Where("assistant_http_logs.time_taken >= ?", ct.GetValue())
			case "<=":
				qry = qry.Where("assistant_http_logs.time_taken <= ?", ct.GetValue())
			}
		case "retry_count":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.retry_count = ?", ct.GetValue())
			case ">=":
				qry = qry.Where("assistant_http_logs.retry_count >= ?", ct.GetValue())
			case "<=":
				qry = qry.Where("assistant_http_logs.retry_count <= ?", ct.GetValue())
			}
		case "status":
			switch ct.GetLogic() {
			case "=":
				qry = qry.Where("assistant_http_logs.status = ?", ct.GetValue())
			}
		}
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					paginate.GetPage(),
					paginate.GetPageSize(),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).Find(&httpLogs)

	if tx.Error != nil {
		s.logger.Errorf("not able to find any HTTP logs %v", tx.Error)
		return cnt, nil, tx.Error
	}

	if order != nil {
		// preserved for forward-compatibility; ordering support to be wired in API phase
	}

	s.logger.Benchmark("assistantHTTPLogService.GetAllLog", time.Since(start))
	return cnt, httpLogs, nil
}

func (s *assistantHTTPLogService) GetLogObject(
	ctx context.Context,
	organizationId uint64,
	projectId uint64,
	httpLogId uint64,
) (requestData []byte, responseData []byte, err error) {
	keyPrefix := s.ObjectPrefix(organizationId, projectId)
	responseKey := s.ObjectKey(keyPrefix, httpLogId, "response.json")
	requestKey := s.ObjectKey(keyPrefix, httpLogId, "request.json")

	type fileStruct struct {
		Data  []byte
		Error error
	}

	responseChan := make(chan fileStruct)
	requestChan := make(chan fileStruct)

	go func(key string) {
		s.logger.Debugf("Getting key from storage %s", key)
		result := s.storage.Get(ctx, key)
		if result.Error != nil {
			s.logger.Errorf("error downloading response: %v", result.Error)
			responseChan <- fileStruct{Error: result.Error}
			close(responseChan)
			return
		}
		responseChan <- fileStruct{Data: result.Data}
		close(responseChan)
	}(responseKey)

	go func(key string) {
		s.logger.Debugf("Getting key from storage %s", key)
		result := s.storage.Get(ctx, key)
		if result.Error != nil {
			s.logger.Errorf("error downloading request: %v", result.Error)
			requestChan <- fileStruct{Error: result.Error}
			close(requestChan)
			return
		}
		requestChan <- fileStruct{Data: result.Data}
		close(requestChan)
	}(requestKey)

	for result := range responseChan {
		if result.Error != nil {
			s.logger.Errorf("error reading response object: %v", result.Error)
			break
		}
		responseData = result.Data
	}

	for result := range requestChan {
		if result.Error != nil {
			s.logger.Errorf("error reading request object: %v", result.Error)
			break
		}
		requestData = result.Data
	}

	return requestData, responseData, nil
}

func (s *assistantHTTPLogService) RetryLog(
	ctx context.Context,
	auth types.SimplePrinciple,
	projectId uint64,
	httpLogId uint64,
) (*internal_assistant_entity.AssistantHTTPLog, error) {
	projectContext, err := types.RequireProject(auth)
	if err != nil {
		return nil, err
	}

	httpLog, err := s.GetLog(ctx, auth, projectId, httpLogId)
	if err != nil {
		return nil, err
	}

	requestPayload, _, _ := s.GetLogObject(ctx, projectContext.OrganizationID, projectContext.ProjectID, httpLogId)
	snapshot, err := s.parseRetryRequestSnapshot(requestPayload)
	if err != nil {
		return nil, fmt.Errorf("http retry: invalid request snapshot for log %d: %w", httpLogId, err)
	}

	url := snapshot.URL
	method := strings.ToUpper(snapshot.Method)
	headers := snapshot.Headers
	timeoutSec := timeoutMsToSeconds(snapshot.TimeoutMs)
	client := rest.NewRestClientWithConfig(url, headers, timeoutSec)
	startTime := time.Now()

	var (
		response *rest.APIResponse
		callErr  error
	)
	switch method {
	case "POST":
		response, callErr = client.Post(ctx, "", snapshot.Body, headers)
	case "PUT":
		response, callErr = client.Put(ctx, "", snapshot.Body, headers)
	case "PATCH":
		response, callErr = client.Patch(ctx, "", snapshot.Body, headers)
	default:
		response, callErr = client.Get(ctx, "", snapshot.Body, headers)
	}

	retryEvent := "retry"
	if httpLog.SourceEvent != "" {
		retryEvent = fmt.Sprintf("retry.%s", httpLog.SourceEvent)
	}

	if callErr != nil {
		errorMessage := callErr.Error()
		return s.CreateLog(
			ctx,
			auth,
			httpLog.Source,
			httpLog.SourceRefId,
			retryEvent,
			httpLog.ContextId,
			httpLog.AssistantId,
			httpLog.AssistantConversationId,
			url,
			method,
			0,
			int64(time.Since(startTime)),
			httpLog.RetryCount+1,
			type_enums.RECORD_FAILED,
			&errorMessage,
			requestPayload,
			nil,
		)
	}

	responsePayload, _ := response.ToJSON()
	status := type_enums.RECORD_COMPLETE
	var errorMessage *string
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		status = type_enums.RECORD_FAILED
		message := fmt.Sprintf("http retry: endpoint returned status %d", response.StatusCode)
		errorMessage = &message
	}

	return s.CreateLog(
		ctx,
		auth,
		httpLog.Source,
		httpLog.SourceRefId,
		retryEvent,
		httpLog.ContextId,
		httpLog.AssistantId,
		httpLog.AssistantConversationId,
		url,
		method,
		int64(response.StatusCode),
		int64(time.Since(startTime)),
		httpLog.RetryCount+1,
		status,
		errorMessage,
		requestPayload,
		responsePayload,
	)
}

func (s *assistantHTTPLogService) parseRetryRequestSnapshot(requestPayload []byte) (retryRequestSnapshot, error) {
	snapshot := retryRequestSnapshot{}
	if len(requestPayload) == 0 {
		return snapshot, fmt.Errorf("empty request payload")
	}
	if err := json.Unmarshal(requestPayload, &snapshot); err != nil {
		return snapshot, err
	}
	if strings.TrimSpace(snapshot.URL) == "" {
		return snapshot, fmt.Errorf("missing request url")
	}
	if strings.TrimSpace(snapshot.Method) == "" {
		return snapshot, fmt.Errorf("missing request method")
	}
	if snapshot.Headers == nil {
		snapshot.Headers = map[string]string{}
	}
	if snapshot.Body == nil {
		snapshot.Body = map[string]interface{}{}
	}
	return snapshot, nil
}

func timeoutMsToSeconds(timeoutMs uint32) uint32 {
	if timeoutMs == 0 {
		return 5
	}
	seconds := timeoutMs / 1000
	if timeoutMs%1000 != 0 {
		seconds++
	}
	if seconds == 0 {
		return 1
	}
	return seconds
}

// Keep storage path unchanged to avoid any request/response asset migration.
func (s *assistantHTTPLogService) ObjectPrefix(orgId, projectId uint64) string {
	return fmt.Sprintf("%d/%d/webhook", orgId, projectId)
}

func (s *assistantHTTPLogService) ObjectKey(prefix string, auditId uint64, objName string) string {
	return fmt.Sprintf("%s/%d__%s", prefix, auditId, objName)
}

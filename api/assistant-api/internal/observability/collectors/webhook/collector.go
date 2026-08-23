// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/clients/rest"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/pkg/utils"
	"github.com/rapidaai/pkg/validator"
	"github.com/rapidaai/protos"
)

const (
	WebhookOptionHTTPMethodKey       = "http_method"
	WebhookOptionHTTPURLKey          = "http_url"
	WebhookOptionHTTPHeadersKey      = "http_headers"
	WebhookOptionRetryStatusCodesKey = "retry_status_codes"
	WebhookOptionMaxRetryCountKey    = "max_retry_count"
	WebhookOptionTimeoutSecondsKey   = "timeout_seconds"

	defaultWebhookTimeoutSeconds uint32 = 60
	minWebhookTimeoutSeconds     uint32 = 1
)

type Config struct {
	Logger                        commons.Logger
	Auth                          *types.Authentication
	AssistantID                   uint64
	AssistantConfigurationService internal_services.AssistantConfigurationService
	HTTPLogService                internal_services.AssistantHTTPLogService
}

type Collector struct {
	logger                        commons.Logger
	auth                          *types.Authentication
	assistantID                   uint64
	assistantConfigurationService internal_services.AssistantConfigurationService
	httpLogService                internal_services.AssistantHTTPLogService
	webhooks                      []*internal_assistant_entity.AssistantConfiguration
	webhooksLoaded                bool
	mu                            sync.Mutex
}

func New(_ context.Context, config Config) observability.Collector {
	if !validator.NonNil(config.Auth) || !validator.NonNil(config.AssistantConfigurationService) || !validator.NonNil(config.HTTPLogService) {
		return observability.NoopCollector{}
	}
	return &Collector{
		logger:                        config.Logger,
		auth:                          config.Auth,
		assistantID:                   config.AssistantID,
		assistantConfigurationService: config.AssistantConfigurationService,
		httpLogService:                config.HTTPLogService,
	}
}

func (c *Collector) Key() string {
	if !validator.NonNil(c) {
		return "webhook"
	}
	return "webhook:" + strconv.FormatUint(c.assistantID, 10)
}

func (c *Collector) Collect(ctx context.Context, scope observability.Scope, _ observability.Context, record observability.Record) error {
	webhookRecord, ok := record.(observability.RecordWebhook)
	if !ok {
		return nil
	}
	if !validator.NonNil(c) {
		return nil
	}
	webhookConfigurations, err := c.webhookConfigurations(ctx)
	if err != nil {
		return err
	}
	if !validator.NotEmpty(webhookConfigurations) {
		return nil
	}
	var webhookErrors []error
	for _, webhookConfiguration := range webhookConfigurations {
		if !c.shouldSend(webhookConfiguration, webhookRecord.Event.String()) {
			continue
		}
		if err := c.send(ctx, scope, webhookConfiguration, webhookRecord.Event.String(), webhookRecord.ContextID, webhookRecord.Payload); err != nil {
			webhookErrors = append(webhookErrors, err)
			if validator.NonNil(c.logger) {
				c.logger.Warnw("observability webhook failed", "webhookID", webhookConfiguration.Id, "event", webhookRecord.Event.String(), "error", err)
			}
		}
	}
	return errors.Join(webhookErrors...)
}

func (c *Collector) Close(context.Context) error {
	return nil
}

func (c *Collector) webhookConfigurations(ctx context.Context) ([]*internal_assistant_entity.AssistantConfiguration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.webhooksLoaded {
		return c.webhooks, nil
	}

	_, webhookConfigurations, err := c.assistantConfigurationService.GetAll(
		ctx,
		c.auth,
		c.assistantID,
		string(internal_assistant_entity.AssistantConfigurationTypeWebhook),
		"",
		nil,
		&protos.Paginate{},
	)
	if err != nil {
		if validator.NonNil(c.logger) {
			c.logger.Warnw("observability webhook load failed", "assistantID", c.assistantID, "error", err)
		}
		return nil, err
	}
	c.webhooks = append([]*internal_assistant_entity.AssistantConfiguration(nil), webhookConfigurations...)
	c.webhooksLoaded = true
	return c.webhooks, nil
}

func (c *Collector) shouldSend(webhookConfiguration *internal_assistant_entity.AssistantConfiguration, eventName string) bool {
	if !validator.NonNil(webhookConfiguration) || !validator.NotBlank(eventName) {
		return false
	}
	if !webhookConfiguration.Enabled {
		return false
	}
	if webhookConfiguration.Provider != "http" {
		return false
	}
	return slices.Contains(webhookConfiguration.GetOptions().GetStringSlice("assistant_events"), eventName)
}

func (c *Collector) send(ctx context.Context, scope observability.Scope, webhookConfiguration *internal_assistant_entity.AssistantConfiguration, webhookEventName string, webhookContextID string, webhookPayload observability.V1WebhookPayload) error {
	webhookOptions := webhookConfiguration.GetOptions()
	webhookHTTPMethod, err := webhookOptions.GetString(WebhookOptionHTTPMethodKey)
	if err != nil || !validator.NotBlank(webhookHTTPMethod) {
		webhookHTTPMethod = http.MethodPost
	}
	webhookHTTPMethod = strings.ToUpper(strings.TrimSpace(webhookHTTPMethod))

	webhookHTTPURL, _ := webhookOptions.GetString(WebhookOptionHTTPURLKey)
	if !validator.NotBlank(webhookHTTPURL) {
		return fmt.Errorf("observability webhook: http_url is required for webhook %d", webhookConfiguration.Id)
	}

	webhookHTTPHeaders, err := webhookOptions.GetStringMap(WebhookOptionHTTPHeadersKey)
	if err != nil {
		webhookHTTPHeaders = map[string]string{}
	}

	webhookMaxRetryCount, err := webhookOptions.GetUint32(WebhookOptionMaxRetryCountKey)
	if err != nil {
		webhookMaxRetryCount = 0
	}

	webhookTimeoutSeconds, err := webhookOptions.GetUint32(WebhookOptionTimeoutSecondsKey)
	if err != nil || !validator.Between(int(webhookTimeoutSeconds), int(minWebhookTimeoutSeconds), int(defaultWebhookTimeoutSeconds)) {
		webhookTimeoutSeconds = defaultWebhookTimeoutSeconds
	}
	webhookRetryStatusCodes := webhookOptions.GetStringSlice(WebhookOptionRetryStatusCodesKey)

	webhookHTTPClient := rest.NewRestClientWithConfig(webhookHTTPURL, webhookHTTPHeaders, webhookTimeoutSeconds)
	var webhookConversationID *uint64
	switch typedScope := scope.(type) {
	case observability.MessageScope:
		scopeConversationID := typedScope.ConversationScopeID()
		webhookConversationID = &scopeConversationID
		if !validator.NotBlank(webhookContextID) {
			webhookContextID = typedScope.ContextID()
		}
	case observability.ConversationScope:
		scopeConversationID := typedScope.ConversationScopeID()
		webhookConversationID = &scopeConversationID
		if !validator.NotBlank(webhookContextID) {
			webhookContextID = typedScope.ContextID()
		}
	case observability.AssistantScope:
		if !validator.NotBlank(webhookContextID) {
			webhookContextID = typedScope.ContextID()
		}
	}
	webhookRequestBody := map[string]interface{}{
		"assistant": map[string]interface{}{
			"id": fmt.Sprintf("%d", c.assistantID),
		},
		"data":  webhookPayload,
		"event": webhookEventName,
	}
	if webhookConversationID != nil && *webhookConversationID != 0 {
		webhookRequestBody["conversation"] = map[string]interface{}{
			"id": fmt.Sprintf("%d", *webhookConversationID),
		}
	}

	for webhookRetryCount := uint32(0); webhookRetryCount <= webhookMaxRetryCount; webhookRetryCount++ {
		webhookAttemptStartTime := time.Now()
		webhookRequestPayload, webhookRequestPayloadMarshalError := json.Marshal(map[string]interface{}{
			"url":        webhookHTTPURL,
			"method":     webhookHTTPMethod,
			"headers":    webhookHTTPHeaders,
			"timeout_ms": webhookTimeoutSeconds * 1000,
			"body":       webhookRequestBody,
		})
		if webhookRequestPayloadMarshalError != nil {
			webhookRequestPayload = nil
		}

		webhookHTTPLogStatus := type_enums.RECORD_COMPLETE
		webhookResponseStatus := int64(0)
		var webhookErrorMessage *string
		var webhookResponsePayload []byte
		var webhookReturnError error
		shouldRetryWebhook := false

		var webhookResponse *rest.APIResponse
		var webhookSendError error
		switch webhookHTTPMethod {
		case http.MethodPut:
			webhookResponse, webhookSendError = webhookHTTPClient.Put(ctx, "", webhookRequestBody, webhookHTTPHeaders)
		case http.MethodPatch:
			webhookResponse, webhookSendError = webhookHTTPClient.Patch(ctx, "", webhookRequestBody, webhookHTTPHeaders)
		case http.MethodGet:
			webhookResponse, webhookSendError = webhookHTTPClient.Get(ctx, "", webhookRequestBody, webhookHTTPHeaders)
		default:
			webhookResponse, webhookSendError = webhookHTTPClient.Post(ctx, "", webhookRequestBody, webhookHTTPHeaders)
		}
		if webhookSendError != nil {
			webhookErrorMessageValue := webhookSendError.Error()
			webhookHTTPLogStatus = type_enums.RECORD_FAILED
			webhookErrorMessage = &webhookErrorMessageValue
			webhookReturnError = webhookSendError
			shouldRetryWebhook = webhookRetryCount < webhookMaxRetryCount
		} else {
			webhookResponseStatus = int64(webhookResponse.StatusCode)
			webhookResponsePayload = webhookResponse.Body
			if utils.MatchAnyString(webhookRetryStatusCodes, strconv.Itoa(webhookResponse.StatusCode)) {
				webhookErrorMessageValue := fmt.Sprintf("observability webhook: retryable status %d", webhookResponse.StatusCode)
				webhookHTTPLogStatus = type_enums.RECORD_FAILED
				webhookErrorMessage = &webhookErrorMessageValue
				webhookReturnError = fmt.Errorf("observability webhook: retryable status %d", webhookResponse.StatusCode)
				shouldRetryWebhook = webhookRetryCount < webhookMaxRetryCount
			} else if webhookResponse.StatusCode < 200 || webhookResponse.StatusCode >= 300 {
				webhookErrorMessageValue := fmt.Sprintf("observability webhook: endpoint returned status %d", webhookResponse.StatusCode)
				webhookHTTPLogStatus = type_enums.RECORD_FAILED
				webhookErrorMessage = &webhookErrorMessageValue
				webhookReturnError = fmt.Errorf("observability webhook: endpoint returned status %d", webhookResponse.StatusCode)
			}
		}

		if _, recordWebhookRequestLogError := c.httpLogService.CreateLog(
			ctx,
			c.auth,
			"webhook",
			webhookConfiguration.Id,
			webhookEventName,
			webhookContextID,
			c.assistantID,
			webhookConversationID,
			webhookHTTPURL,
			webhookHTTPMethod,
			webhookResponseStatus,
			int64(time.Since(webhookAttemptStartTime)),
			webhookRetryCount,
			webhookHTTPLogStatus,
			webhookErrorMessage,
			webhookRequestPayload,
			webhookResponsePayload,
		); recordWebhookRequestLogError != nil && validator.NonNil(c.logger) {
			c.logger.Warnw("observability webhook request log record failed", "webhookID", webhookConfiguration.Id, "event", webhookEventName, "error", recordWebhookRequestLogError)
		}

		if shouldRetryWebhook {
			time.Sleep(2 * time.Second)
			continue
		}
		return webhookReturnError
	}
	return nil
}

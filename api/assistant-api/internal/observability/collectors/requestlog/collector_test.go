// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package observability_collector_requestlog

import (
	"context"
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	"github.com/rapidaai/api/assistant-api/internal/observability"
	"github.com/rapidaai/pkg/authentication"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestCollectorCreatesRequestLog(t *testing.T) {
	httpLogService := &recordingHTTPLogService{}
	collector := New(Config{HTTPLogService: httpLogService})
	auth, err := authentication.NewDirectPrincipal(
		authentication.Identity{Type: authentication.IdentityService, ID: 7, Issuer: "assistant-api"},
		authentication.Tenant{OrganizationID: 11, ProjectID: 13},
		authentication.Proof{
			Method:   authentication.AuthenticationServiceAssertion,
			Audience: authentication.AudienceRapidaInternal,
			Reference: authentication.ProofReference{
				Type:  authentication.ProofReferenceSigningKey,
				Value: "assistant-test-1",
			},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("NewDirectPrincipal() error = %v", err)
	}

	err = collector.Collect(
		context.Background(),
		observability.ConversationScope{
			AssistantScope: observability.AssistantScope{AssistantID: 10},
			ConversationID: 20,
		},
		observability.Context{Auth: auth},
		observability.RecordRequestLog{
			Source:         "webhook",
			SourceRefID:    30,
			SourceEvent:    "call.ringing",
			ContextID:      "20",
			HTTPURL:        "https://example.com/webhook",
			HTTPMethod:     "POST",
			ResponseStatus: httpStatusNoContent,
			TimeTaken:      40,
			RetryCount:     1,
			Status:         type_enums.RECORD_COMPLETE,
			RequestPayload: []byte(`{"request":true}`),
		},
	)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}
	if len(httpLogService.calls) != 1 {
		t.Fatalf("expected one request log call, got %d", len(httpLogService.calls))
	}

	requestLogCall := httpLogService.calls[0]
	if requestLogCall.source != "webhook" || requestLogCall.sourceRefID != 30 || requestLogCall.sourceEvent != "call.ringing" {
		t.Fatalf("unexpected source fields: %+v", requestLogCall)
	}
	if requestLogCall.assistantID != 10 || requestLogCall.conversationID == nil || *requestLogCall.conversationID != 20 {
		t.Fatalf("unexpected scope fields: %+v", requestLogCall)
	}
	if requestLogCall.status != type_enums.RECORD_COMPLETE || requestLogCall.responseStatus != httpStatusNoContent {
		t.Fatalf("unexpected status fields: %+v", requestLogCall)
	}
}

const httpStatusNoContent int64 = 204

type requestLogCall struct {
	source         string
	sourceRefID    uint64
	sourceEvent    string
	assistantID    uint64
	conversationID *uint64
	responseStatus int64
	status         type_enums.RecordState
}

type recordingHTTPLogService struct {
	calls []requestLogCall
}

func (s *recordingHTTPLogService) CreateLog(
	_ context.Context,
	_ authentication.Principal,
	source string,
	sourceRefID uint64,
	sourceEvent string,
	_ string,
	assistantID uint64,
	conversationID *uint64,
	_ string,
	_ string,
	responseStatus int64,
	_ int64,
	_ uint32,
	status type_enums.RecordState,
	_ *string,
	_ []byte,
	_ []byte,
) (*internal_assistant_entity.AssistantHTTPLog, error) {
	s.calls = append(s.calls, requestLogCall{
		source:         source,
		sourceRefID:    sourceRefID,
		sourceEvent:    sourceEvent,
		assistantID:    assistantID,
		conversationID: conversationID,
		responseStatus: responseStatus,
		status:         status,
	})
	return &internal_assistant_entity.AssistantHTTPLog{}, nil
}

func (s *recordingHTTPLogService) GetLog(context.Context, authentication.Principal, uint64, uint64) (*internal_assistant_entity.AssistantHTTPLog, error) {
	return nil, nil
}

func (s *recordingHTTPLogService) GetAllLog(context.Context, authentication.Principal, uint64, []*protos.Criteria, *protos.Paginate, *protos.Ordering) (int64, []*internal_assistant_entity.AssistantHTTPLog, error) {
	return 0, nil, nil
}

func (s *recordingHTTPLogService) GetLogObject(context.Context, authentication.Principal, uint64) ([]byte, []byte, error) {
	return nil, nil, nil
}

func (s *recordingHTTPLogService) RetryLog(context.Context, authentication.Principal, uint64, uint64) (*internal_assistant_entity.AssistantHTTPLog, error) {
	return nil, nil
}

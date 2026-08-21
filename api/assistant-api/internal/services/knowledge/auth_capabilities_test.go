package internal_knowledge_service

import (
	"context"
	"testing"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
)

func projectAuth(organizationID, projectID uint64) *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeProject,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}
}

func organizationAuth(organizationID uint64) *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
	}
}

func userAuth(userID, organizationID, projectID uint64) *types.Authentication {
	return &types.Authentication{
		AuthType:          types.AuthTypeUser,
		UserValue:         &types.UserContext{UserID: userID},
		OrganizationValue: &types.OrganizationContext{OrganizationID: organizationID},
		ProjectValue:      &types.ProjectContext{OrganizationID: organizationID, ProjectID: projectID},
	}
}

func TestKnowledgeReadsRequireProjectContext(t *testing.T) {
	auth := organizationAuth(10)
	service := &knowledgeService{}
	documentService := &knowledgeDocumentService{}

	if _, _, err := service.GetAll(context.Background(), auth, nil, &protos.Paginate{}); err == nil {
		t.Fatal("GetAll() error = nil without project context")
	}
	if _, err := service.Get(context.Background(), auth, 1); err == nil {
		t.Fatal("Get() error = nil without project context")
	}
	if _, _, err := documentService.GetAll(context.Background(), auth, 1, nil, &protos.Paginate{}); err == nil {
		t.Fatal("document GetAll() error = nil without project context")
	}
	if _, err := documentService.Get(context.Background(), auth, 1, 2); err == nil {
		t.Fatal("document Get() error = nil without project context")
	}
	documentCount, wordCount, tokenCount := documentService.GetCounts(context.Background(), auth, 1)
	if documentCount != 0 || wordCount != 0 || tokenCount != 0 {
		t.Fatalf("GetCounts() = %d, %d, %d", documentCount, wordCount, tokenCount)
	}
}

func TestKnowledgeAuditMutationsRequireUserIdentity(t *testing.T) {
	auth := projectAuth(10, 20)
	service := &knowledgeService{}
	documentService := &knowledgeDocumentService{}

	if _, err := service.CreateOrUpdateKnowledgeTag(context.Background(), auth, 1, []string{"tag"}); err == nil {
		t.Fatal("CreateOrUpdateKnowledgeTag() error = nil without user identity")
	}
	if _, err := service.CreateKnowledge(context.Background(), auth, "knowledge", nil, nil, "provider", nil); err == nil {
		t.Fatal("CreateKnowledge() error = nil without user identity")
	}
	if _, err := service.UpdateKnowledgeDetail(context.Background(), auth, 1, "name", "description"); err == nil {
		t.Fatal("UpdateKnowledgeDetail() error = nil without user identity")
	}
	if _, err := documentService.CreateToolDocument(context.Background(), auth, nil, "tool", "plain", nil); err == nil {
		t.Fatal("CreateToolDocument() error = nil without user identity")
	}
	if _, err := documentService.CreateManualDocument(context.Background(), auth, nil, "manual-url", "plain", nil); err == nil {
		t.Fatal("CreateManualDocument() error = nil without user identity")
	}
}

func TestKnowledgeLogReadsRejectProjectMismatch(t *testing.T) {
	auth := userAuth(1, 10, 20)
	service := &knowledgeService{}

	if _, err := service.GetLog(context.Background(), auth, 21, 1); err == nil {
		t.Fatal("GetLog() error = nil for mismatched project")
	}
	if _, _, err := service.GetAllLog(context.Background(), auth, 21, nil, &protos.Paginate{}, nil); err == nil {
		t.Fatal("GetAllLog() error = nil for mismatched project")
	}
}

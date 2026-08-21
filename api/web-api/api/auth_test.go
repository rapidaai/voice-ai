package web_api

import (
	"context"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

func TestScopeAuthorizeIncludesProjectActor(t *testing.T) {
	actor := types.ActorIdentity{Type: types.ActorTypeProject, ID: "55"}
	auth := &types.Authentication{
		AuthType:          types.AuthTypeProject,
		ActorValue:        &actor,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 99},
		ProjectValue:      &types.ProjectContext{OrganizationID: 99, ProjectID: 77},
	}
	ctx := context.WithValue(context.Background(), types.CTX_, auth)

	response, err := (&webAuthGRPCApi{}).ScopeAuthorize(ctx, &protos.ScopeAuthorizeRequest{Scope: "project"})
	if err != nil {
		t.Fatalf("ScopeAuthorize() error = %v", err)
	}
	data := response.GetData()
	if data.GetActorType() != "project" || data.GetActorId() != "55" {
		t.Fatalf("ScopeAuthorize() actor = %q/%q", data.GetActorType(), data.GetActorId())
	}
	if data.GetProjectId() != 77 || data.GetOrganizationId() != 99 {
		t.Fatalf("ScopeAuthorize() scope = %d/%d", data.GetProjectId(), data.GetOrganizationId())
	}
}

func TestScopedAuthenticationActorFieldNumbers(t *testing.T) {
	descriptor := (&protos.ScopedAuthentication{}).ProtoReflect().Descriptor()
	actorType := descriptor.Fields().ByNumber(5)
	actorID := descriptor.Fields().ByNumber(6)
	if actorType == nil || string(actorType.Name()) != "actorType" || !actorType.HasOptionalKeyword() {
		t.Fatalf("field 5 = %v", actorType)
	}
	if actorID == nil || string(actorID.Name()) != "actorId" || !actorID.HasOptionalKeyword() {
		t.Fatalf("field 6 = %v", actorID)
	}
	for number := 1; number <= 4; number++ {
		if descriptor.Fields().ByNumber(protoreflect.FieldNumber(number)) == nil {
			t.Fatalf("existing field %d is missing", number)
		}
	}
}

func TestScopeAuthorizeRejectsWrongScope(t *testing.T) {
	auth := &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		OrganizationValue: &types.OrganizationContext{OrganizationID: 99},
	}
	ctx := context.WithValue(context.Background(), types.CTX_, auth)
	if _, err := (&webAuthGRPCApi{}).ScopeAuthorize(ctx, &protos.ScopeAuthorizeRequest{Scope: "project"}); err == nil {
		t.Fatal("ScopeAuthorize() error = nil")
	}
}

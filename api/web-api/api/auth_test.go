package web_api

import (
	"context"
	"math"
	"strconv"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

func TestScopeAuthorizeIncludesProjectActor(t *testing.T) {
	actor := types.ActorIdentity{Type: types.ActorTypeProject, ID: 55}
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

func TestScopeAuthorizeIncludesOrganizationCredentialActor(t *testing.T) {
	auth := &types.Authentication{
		AuthType:          types.AuthTypeOrg,
		ActorValue:        &types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 56},
		OrganizationValue: &types.OrganizationContext{OrganizationID: 99},
	}
	ctx := context.WithValue(context.Background(), types.CTX_, auth)
	response, err := (&webAuthGRPCApi{}).ScopeAuthorize(ctx, &protos.ScopeAuthorizeRequest{Scope: "organization"})
	if err != nil {
		t.Fatalf("ScopeAuthorize() error = %v", err)
	}
	data := response.GetData()
	if data.GetActorType() != "organization" || data.GetActorId() != "56" {
		t.Fatalf("ScopeAuthorize() actor = %q/%q", data.GetActorType(), data.GetActorId())
	}
	if data.GetOrganizationId() != 99 || data.GetProjectId() != 0 {
		t.Fatalf("ScopeAuthorize() scope = %d/%d", data.GetOrganizationId(), data.GetProjectId())
	}
}

func TestScopeAuthorizeActorRange(t *testing.T) {
	tests := []struct {
		name      string
		actorID   uint64
		wantError bool
	}{
		{name: "zero rejected", actorID: 0, wantError: true},
		{name: "max bigint accepted", actorID: math.MaxInt64},
		{name: "above max bigint rejected", actorID: uint64(math.MaxInt64) + 1, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			auth := &types.Authentication{
				AuthType:          types.AuthTypeProject,
				ActorValue:        &types.ActorIdentity{Type: types.ActorTypeProject, ID: test.actorID},
				OrganizationValue: &types.OrganizationContext{OrganizationID: 99},
				ProjectValue:      &types.ProjectContext{OrganizationID: 99, ProjectID: 77},
			}
			ctx := context.WithValue(context.Background(), types.CTX_, auth)

			response, err := (&webAuthGRPCApi{}).ScopeAuthorize(ctx, &protos.ScopeAuthorizeRequest{Scope: "project"})
			if test.wantError {
				if err == nil {
					t.Fatal("ScopeAuthorize() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ScopeAuthorize() error = %v", err)
			}
			if response.GetData().GetActorId() != strconv.FormatUint(test.actorID, 10) {
				t.Fatalf("ScopeAuthorize() actor ID = %q", response.GetData().GetActorId())
			}
		})
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

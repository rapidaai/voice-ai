package web_api

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/rapidaai/pkg/types"
	protos "github.com/rapidaai/protos"
)

func TestScopedAuthenticationIncludesProjectActor(t *testing.T) {
	credentialID := uint64(55)
	projectID := uint64(77)
	organizationID := uint64(99)
	principle := &types.ProjectScope{
		CredentialId:   &credentialID,
		ProjectId:      &projectID,
		OrganizationId: &organizationID,
		Status:         "ACTIVE",
	}

	auth, err := scopedAuthentication(principle)
	if err != nil {
		t.Fatalf("scopedAuthentication() error = %v", err)
	}
	if auth.GetActorType() != "project" || auth.GetActorId() != "55" {
		t.Fatalf("scopedAuthentication() actor = %q/%q", auth.GetActorType(), auth.GetActorId())
	}
	if auth.GetProjectId() != projectID || auth.GetOrganizationId() != organizationID {
		t.Fatalf("scopedAuthentication() scope = %d/%d", auth.GetProjectId(), auth.GetOrganizationId())
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

func TestScopedAuthenticationRejectsMissingProjectCredential(t *testing.T) {
	projectID := uint64(77)
	organizationID := uint64(99)
	principle := &types.ProjectScope{
		ProjectId:      &projectID,
		OrganizationId: &organizationID,
		Status:         "ACTIVE",
	}
	if _, err := scopedAuthentication(principle); err == nil {
		t.Fatal("scopedAuthentication() error = nil, want missing actor error")
	}
}

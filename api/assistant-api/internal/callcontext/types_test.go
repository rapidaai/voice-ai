package internal_callcontext

import (
	"testing"

	"github.com/rapidaai/pkg/types"
)

func TestCallContextToAuthIncludesServiceScope(t *testing.T) {
	callContext := &CallContext{
		AuthToken:      "service-token",
		ProjectID:      22,
		OrganizationID: 33,
	}

	auth := callContext.ToAuth()
	if auth.GetCurrentToken() != "service-token" {
		t.Fatalf("expected auth token service-token, got %q", auth.GetCurrentToken())
	}
	delegatedContext, err := types.ResolveDelegatedContext(auth)
	if err != nil {
		t.Fatalf("ResolveDelegatedContext() error = %v", err)
	}
	if delegatedContext.ProjectID == nil || *delegatedContext.ProjectID != 22 || delegatedContext.OrganizationID != 33 {
		t.Fatalf("unexpected delegated context: %+v", delegatedContext)
	}
}

func TestCallContextToAuthOmitsEmptyScopeIDs(t *testing.T) {
	callContext := &CallContext{AuthToken: "service-token"}

	auth := callContext.ToAuth()
	if auth.GetCurrentToken() != "service-token" {
		t.Fatalf("expected auth token service-token, got %q", auth.GetCurrentToken())
	}
	if _, err := types.ResolveDelegatedContext(auth); err == nil {
		t.Fatal("ResolveDelegatedContext() error = nil, want missing organization error")
	}
}

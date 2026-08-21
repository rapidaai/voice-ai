package types

import (
	"testing"

	type_enums "github.com/rapidaai/pkg/types/enums"
)

type authenticationOnlyStub struct {
	authenticated bool
}

func (stub authenticationOnlyStub) IsAuthenticated() bool   { return stub.authenticated }
func (stub authenticationOnlyStub) GetCurrentToken() string { return "token" }
func (stub authenticationOnlyStub) Type() AuthType          { return AuthTypeUser }

type malformedCapabilityStub struct {
	authenticationOnlyStub
}

func (malformedCapabilityStub) UserIdentity() (uint64, bool)        { return 0, true }
func (malformedCapabilityStub) OrganizationContext() (uint64, bool) { return 0, true }
func (malformedCapabilityStub) ProjectContext() (ProjectContext, bool) {
	return ProjectContext{OrganizationID: 1}, true
}
func (malformedCapabilityStub) DelegatedContext() (DelegatedContext, bool) {
	zero := uint64(0)
	return DelegatedContext{OrganizationID: 1, UserID: &zero}, true
}

type derivedContextStub struct {
	authenticationOnlyStub
	organizationID       uint64
	organizationProvided bool
	userID               uint64
	userProvided         bool
	projectContext       ProjectContext
	projectProvided      bool
}

type delegatedOnlyStub struct {
	authenticationOnlyStub
	context  DelegatedContext
	provided bool
}

func (stub delegatedOnlyStub) DelegatedContext() (DelegatedContext, bool) {
	return stub.context, stub.provided
}

func (stub derivedContextStub) OrganizationContext() (uint64, bool) {
	return stub.organizationID, stub.organizationProvided
}

func (stub derivedContextStub) UserIdentity() (uint64, bool) {
	return stub.userID, stub.userProvided
}

func (stub derivedContextStub) ProjectContext() (ProjectContext, bool) {
	return stub.projectContext, stub.projectProvided
}

func TestRequireCapabilitiesFailClosed(t *testing.T) {
	var typedNilUser *PlainAuthPrinciple
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "nil user", run: func() error { _, err := RequireUser(nil); return err }},
		{name: "typed nil user", run: func() error { _, err := RequireUser(typedNilUser); return err }},
		{name: "unauthenticated organization", run: func() error { _, err := RequireOrganization(authenticationOnlyStub{}); return err }},
		{name: "missing project provider", run: func() error { _, err := RequireProject(authenticationOnlyStub{authenticated: true}); return err }},
		{name: "malformed user", run: func() error {
			_, err := RequireUser(malformedCapabilityStub{authenticationOnlyStub{authenticated: true}})
			return err
		}},
		{name: "malformed organization", run: func() error {
			_, err := RequireOrganization(malformedCapabilityStub{authenticationOnlyStub{authenticated: true}})
			return err
		}},
		{name: "malformed project", run: func() error {
			_, err := RequireProject(malformedCapabilityStub{authenticationOnlyStub{authenticated: true}})
			return err
		}},
		{name: "malformed delegation", run: func() error {
			_, err := ResolveDelegatedContext(malformedCapabilityStub{authenticationOnlyStub{authenticated: true}})
			return err
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err == nil {
				t.Fatal("error = nil")
			}
		})
	}
}

func TestSimplePrincipleAliasesAuthenticationPrinciple(t *testing.T) {
	var simple SimplePrinciple = authenticationOnlyStub{authenticated: true}
	var authentication AuthenticationPrinciple = simple
	if !authentication.IsAuthenticated() {
		t.Fatal("AuthenticationPrinciple alias lost authentication behavior")
	}
}

func TestRequireProjectRejectsMissingOrMalformedDelegatedProject(t *testing.T) {
	zero := uint64(0)
	tests := []struct {
		name string
		auth AuthenticationPrinciple
	}{
		{
			name: "missing project",
			auth: delegatedOnlyStub{
				authenticationOnlyStub: authenticationOnlyStub{authenticated: true},
				context:                DelegatedContext{OrganizationID: 1},
				provided:               true,
			},
		},
		{
			name: "zero project",
			auth: delegatedOnlyStub{
				authenticationOnlyStub: authenticationOnlyStub{authenticated: true},
				context:                DelegatedContext{OrganizationID: 1, ProjectID: &zero},
				provided:               true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := RequireProject(test.auth); err == nil {
				t.Fatal("RequireProject() error = nil")
			}
		})
	}
}

func TestResolveDelegatedContextDerivesCompatibleCapabilities(t *testing.T) {
	user := &PlainAuthPrinciple{
		User:               UserInfo{Id: 7},
		OrganizationRole:   &OrganizaitonRole{OrganizationId: 8},
		CurrentProjectRole: &ProjectRole{ProjectId: 9},
	}
	projectID := uint64(12)
	projectOrganizationID := uint64(11)
	project := &ProjectScope{
		ProjectId:      &projectID,
		OrganizationId: &projectOrganizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}
	organizationID := uint64(13)
	organization := &OrganizationScope{
		OrganizationId: &organizationID,
		Status:         type_enums.RECORD_ACTIVE.String(),
	}

	tests := []struct {
		name         string
		auth         AuthenticationPrinciple
		organization uint64
		user         *uint64
		project      *uint64
	}{
		{name: "user", auth: user, organization: 8, user: uint64Pointer(7), project: uint64Pointer(9)},
		{name: "project", auth: project, organization: 11, project: uint64Pointer(12)},
		{name: "organization", auth: organization, organization: 13},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			delegatedContext, err := ResolveDelegatedContext(test.auth)
			if err != nil {
				t.Fatalf("ResolveDelegatedContext() error = %v", err)
			}
			if delegatedContext.OrganizationID != test.organization {
				t.Fatalf("OrganizationID = %d, want %d", delegatedContext.OrganizationID, test.organization)
			}
			assertOptionalID(t, "UserID", delegatedContext.UserID, test.user)
			assertOptionalID(t, "ProjectID", delegatedContext.ProjectID, test.project)
		})
	}
}

func TestResolveDelegatedContextRejectsMalformedDerivedCapabilities(t *testing.T) {
	tests := []struct {
		name string
		auth AuthenticationPrinciple
	}{
		{
			name: "missing organization provider",
			auth: authenticationOnlyStub{authenticated: true},
		},
		{
			name: "malformed organization",
			auth: derivedContextStub{
				authenticationOnlyStub: authenticationOnlyStub{authenticated: true},
				organizationProvided:   true,
			},
		},
		{
			name: "malformed user",
			auth: derivedContextStub{
				authenticationOnlyStub: authenticationOnlyStub{authenticated: true},
				organizationID:         1,
				organizationProvided:   true,
				userProvided:           true,
			},
		},
		{
			name: "malformed project",
			auth: derivedContextStub{
				authenticationOnlyStub: authenticationOnlyStub{authenticated: true},
				organizationID:         1,
				organizationProvided:   true,
				projectContext:         ProjectContext{OrganizationID: 1},
				projectProvided:        true,
			},
		},
		{
			name: "project organization mismatch",
			auth: derivedContextStub{
				authenticationOnlyStub: authenticationOnlyStub{authenticated: true},
				organizationID:         1,
				organizationProvided:   true,
				projectContext:         ProjectContext{OrganizationID: 2, ProjectID: 3},
				projectProvided:        true,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ResolveDelegatedContext(test.auth); err == nil {
				t.Fatal("ResolveDelegatedContext() error = nil")
			}
		})
	}
}

func uint64Pointer(value uint64) *uint64 {
	return &value
}

func assertOptionalID(t *testing.T, name string, got *uint64, want *uint64) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Fatalf("%s = %d, want nil", name, *got)
		}
		return
	}
	if got == nil || *got != *want {
		t.Fatalf("%s = %v, want %d", name, got, *want)
	}
}

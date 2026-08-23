package authenticators

import (
	"context"
	"math"
	"strconv"
	"testing"

	"github.com/rapidaai/config"
	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
	web_api "github.com/rapidaai/protos"
)

type organizationAuthClient struct {
	response *web_api.ScopedAuthentication
}

func (c *organizationAuthClient) Authorize(context.Context, string, uint64) (*web_api.Authentication, error) {
	return nil, nil
}
func (c *organizationAuthClient) ScopeAuthorize(context.Context, string, string) (*web_api.ScopedAuthentication, error) {
	return c.response, nil
}

var _ web_client.AuthClient = (*organizationAuthClient)(nil)

func TestOrganizationAuthenticatorPropagatesCredentialActor(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	actorType := "organization"
	actorID := "42"
	authenticator := NewOrganizationAuthenticator(&config.AppConfig{}, logger, &organizationAuthClient{response: &web_api.ScopedAuthentication{
		OrganizationId: 9,
		Status:         "ACTIVE",
		ActorType:      &actorType,
		ActorId:        &actorID,
	}})
	principle, err := authenticator.Claim(context.Background(), "secret")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	actor, err := types.ResolveAuditActor(principle.Info)
	if err != nil || actor != (types.ActorIdentity{Type: types.ActorTypeOrganization, ID: 42}) {
		t.Fatalf("ResolveAuditActor() = %+v, %v", actor, err)
	}
}

func TestOrganizationAuthenticatorRejectsInvalidActor(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	organizationType := "organization"
	projectType := "project"
	zero := "0"
	max := strconv.FormatUint(math.MaxInt64, 10)
	aboveMax := strconv.FormatUint(uint64(math.MaxInt64)+1, 10)
	tests := []struct {
		name      string
		actorType *string
		actorID   *string
		wantError bool
	}{
		{name: "missing", wantError: true},
		{name: "zero", actorType: &organizationType, actorID: &zero, wantError: true},
		{name: "mismatched", actorType: &projectType, actorID: &max, wantError: true},
		{name: "max bigint", actorType: &organizationType, actorID: &max},
		{name: "above max", actorType: &organizationType, actorID: &aboveMax, wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authenticator := NewOrganizationAuthenticator(&config.AppConfig{}, logger, &organizationAuthClient{response: &web_api.ScopedAuthentication{
				OrganizationId: 9,
				Status:         "ACTIVE",
				ActorType:      test.actorType,
				ActorId:        test.actorID,
			}})
			principle, err := authenticator.Claim(context.Background(), "secret")
			if test.wantError {
				if err == nil {
					t.Fatal("Claim() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Claim() error = %v", err)
			}
			actor, err := types.ResolveAuditActor(principle.Info)
			if err != nil || actor.ID != math.MaxInt64 {
				t.Fatalf("ResolveAuditActor() = %+v, %v", actor, err)
			}
		})
	}
}

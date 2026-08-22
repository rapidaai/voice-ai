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

type projectAuthClient struct {
	response *web_api.ScopedAuthentication
}

func (c *projectAuthClient) Authorize(context.Context, string, uint64) (*web_api.Authentication, error) {
	return nil, nil
}
func (c *projectAuthClient) ScopeAuthorize(context.Context, string, string) (*web_api.ScopedAuthentication, error) {
	return c.response, nil
}

var _ web_client.AuthClient = (*projectAuthClient)(nil)

func TestProjectAuthenticatorPropagatesCredentialActor(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	actorType := "project"
	actorID := "42"
	authenticator := NewProjectAuthenticator(&config.AppConfig{}, logger, &projectAuthClient{response: &web_api.ScopedAuthentication{
		OrganizationId: 1,
		ProjectId:      2,
		Status:         "ACTIVE",
		ActorType:      &actorType,
		ActorId:        &actorID,
	}})

	principle, err := authenticator.Claim(context.Background(), "secret")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	actor, ok := principle.Info.AuditActor()
	if !ok || actor.ID != 42 {
		t.Fatalf("AuditActor() = %+v, %v", actor, ok)
	}
}

func TestProjectAuthenticatorActorRange(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	projectActorType := "project"
	userActorType := "user"
	zeroActorID := "0"
	maxActorID := strconv.FormatUint(math.MaxInt64, 10)
	aboveMaxActorID := strconv.FormatUint(uint64(math.MaxInt64)+1, 10)
	tests := []struct {
		name      string
		actorType *string
		actorID   *string
		wantID    uint64
		wantError bool
	}{
		{name: "missing rejected", wantError: true},
		{name: "zero rejected", actorType: &projectActorType, actorID: &zeroActorID, wantError: true},
		{name: "mismatched type rejected", actorType: &userActorType, actorID: &maxActorID, wantError: true},
		{name: "max bigint accepted", actorType: &projectActorType, actorID: &maxActorID, wantID: math.MaxInt64},
		{name: "above max bigint rejected", actorType: &projectActorType, actorID: &aboveMaxActorID, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authenticator := NewProjectAuthenticator(&config.AppConfig{}, logger, &projectAuthClient{response: &web_api.ScopedAuthentication{
				OrganizationId: 1,
				ProjectId:      2,
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
			if err != nil || actor.ID != test.wantID {
				t.Fatalf("ResolveAuditActor() = %+v, %v", actor, err)
			}
		})
	}
}

func TestProjectAuthenticatorRejectsMalformedActor(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	actorType := "user"
	actorID := "not-a-number"
	authenticator := NewProjectAuthenticator(&config.AppConfig{}, logger, &projectAuthClient{response: &web_api.ScopedAuthentication{
		ActorType: &actorType,
		ActorId:   &actorID,
	}})
	if _, err := authenticator.Claim(context.Background(), "secret"); err == nil {
		t.Fatal("Claim() error = nil, want malformed actor error")
	}
}

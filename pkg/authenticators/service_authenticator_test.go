package authenticators

import (
	"context"
	"testing"
	"time"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

func TestServiceAuthenticatorUsesSharedJWTSecret(t *testing.T) {
	logger, err := commons.NewApplicationLogger(commons.EnableConsole(true), commons.EnableFile(false))
	if err != nil {
		t.Fatal(err)
	}
	token, err := types.CreateServiceScopeToken(types.DelegatedContext{OrganizationID: 7}, types.ServiceAssertion{
		ActorID: 41,
		Issuer:  "assistant-api",
		TTL:     time.Minute,
	}, "shared-secret")
	if err != nil {
		t.Fatal(err)
	}

	authenticator := NewServiceAuthenticator(&config.AppConfig{Secret: "shared-secret"}, logger)
	principle, err := authenticator.Claim(context.Background(), token)
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if principle.Info.ActorId != 41 {
		t.Fatalf("Claim() actor ID = %d", principle.Info.ActorId)
	}

	rejected := NewServiceAuthenticator(&config.AppConfig{Secret: "wrong-secret"}, logger)
	if _, err := rejected.Claim(context.Background(), token); err == nil {
		t.Fatal("Claim() error = nil for wrong shared secret")
	}
}

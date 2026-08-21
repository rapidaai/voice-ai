package types

import "testing"

type actorIdentityTestPrinciple struct {
	actor         ActorIdentity
	authenticated bool
	available     bool
}

func (p *actorIdentityTestPrinciple) IsAuthenticated() bool             { return p.authenticated }
func (p *actorIdentityTestPrinciple) GetCurrentToken() string           { return "" }
func (p *actorIdentityTestPrinciple) Type() AuthType                    { return AuthTypeProject }
func (p *actorIdentityTestPrinciple) AuditActor() (ActorIdentity, bool) { return p.actor, p.available }

type actorIdentityUnsupportedPrinciple struct{}

func (p *actorIdentityUnsupportedPrinciple) IsAuthenticated() bool   { return true }
func (p *actorIdentityUnsupportedPrinciple) GetCurrentToken() string { return "" }
func (p *actorIdentityUnsupportedPrinciple) Type() AuthType          { return AuthTypeProject }

func TestResolveAuditActor(t *testing.T) {
	tests := []struct {
		name      string
		principle SimplePrinciple
		want      ActorIdentity
		wantError bool
	}{
		{
			name: "resolves durable actor",
			principle: &actorIdentityTestPrinciple{
				actor:         ActorIdentity{Type: ActorTypeProject, ID: "42"},
				authenticated: true,
				available:     true,
			},
			want: ActorIdentity{Type: ActorTypeProject, ID: "42"},
		},
		{
			name:      "rejects unauthenticated principle",
			principle: &actorIdentityTestPrinciple{},
			wantError: true,
		},
		{
			name:      "rejects unsupported principle",
			principle: &actorIdentityUnsupportedPrinciple{},
			wantError: true,
		},
		{
			name: "rejects unavailable actor",
			principle: &actorIdentityTestPrinciple{
				authenticated: true,
			},
			wantError: true,
		},
		{
			name: "rejects unknown actor",
			principle: &actorIdentityTestPrinciple{
				actor:         ActorIdentity{Type: ActorTypeUnknown, ID: "42"},
				authenticated: true,
				available:     true,
			},
			wantError: true,
		},
		{
			name: "rejects empty actor type",
			principle: &actorIdentityTestPrinciple{
				actor:         ActorIdentity{ID: "42"},
				authenticated: true,
				available:     true,
			},
			wantError: true,
		},
		{
			name: "rejects malformed actor type",
			principle: &actorIdentityTestPrinciple{
				actor:         ActorIdentity{Type: ActorType("invalid"), ID: "42"},
				authenticated: true,
				available:     true,
			},
			wantError: true,
		},
		{
			name: "rejects empty actor id",
			principle: &actorIdentityTestPrinciple{
				actor:         ActorIdentity{Type: ActorTypeProject},
				authenticated: true,
				available:     true,
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveAuditActor(test.principle)
			if test.wantError {
				if err == nil {
					t.Fatal("ResolveAuditActor() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveAuditActor() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("ResolveAuditActor() = %+v, want %+v", got, test.want)
			}
		})
	}
}

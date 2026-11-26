// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_authenticators

import (
	"testing"

	"github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
)

func TestPrincipalResolverConstructorsBuildConfiguredMethods(t *testing.T) {
	postgres := connectors.NewPostgresConnector(&configs.PostgresConfig{}, nil)
	constructors := []struct {
		name string
		new  func() any
	}{
		{name: "user", new: func() any { return NewUserResolver(nil, postgres) }},
		{name: "project", new: func() any { return NewProjectResolver(nil, postgres) }},
		{name: "organization", new: func() any { return NewOrganizationResolver(make([]byte, 32), nil, postgres) }},
	}
	for _, constructor := range constructors {
		t.Run(constructor.name, func(t *testing.T) {
			if principalResolver := constructor.new(); principalResolver == nil {
				t.Fatal("constructor returned nil resolver")
			}
		})
	}
}

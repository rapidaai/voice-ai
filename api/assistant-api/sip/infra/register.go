// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package sip_infra

import (
	"context"

	"github.com/emiago/sipgo"
	internal_core "github.com/rapidaai/api/assistant-api/sip/internal/core"
	"github.com/rapidaai/pkg/commons"
)

func NewRegistrationClient(client *sipgo.Client, listenConfig *ListenConfig, logger commons.Logger) *RegistrationClient {
	return &RegistrationClient{
		inner: internal_core.NewRegistrationClient(client, listenConfig, logger),
	}
}

func (rc *RegistrationClient) Register(ctx context.Context, reg *Registration) error {
	return rc.inner.Register(ctx, reg)
}

func (rc *RegistrationClient) SetObserver(observer RegistrationObserver) {
	rc.inner.SetObserver(observer)
}

func (rc *RegistrationClient) Unregister(ctx context.Context, did string) error {
	return rc.inner.Unregister(ctx, did)
}

func (rc *RegistrationClient) UnregisterAll(ctx context.Context) {
	rc.inner.UnregisterAll(ctx)
}

func (rc *RegistrationClient) IsRegistered(did string) bool {
	return rc.inner.IsRegistered(did)
}

func (rc *RegistrationClient) Snapshot(did string) RegistrationSnapshot {
	return rc.inner.Snapshot(did)
}

func (rc *RegistrationClient) ActiveCount() int {
	return rc.inner.ActiveCount()
}

func (rc *RegistrationClient) GetRegisteredDIDs() []string {
	return rc.inner.GetRegisteredDIDs()
}

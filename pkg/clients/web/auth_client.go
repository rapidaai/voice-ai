// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package web_client

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/rapidaai/config"
	"github.com/rapidaai/pkg/authentication"
	"github.com/rapidaai/pkg/authentication/resolver"
	"github.com/rapidaai/pkg/clients"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	"github.com/rapidaai/protos"
)

// AuthenticationClient resolves opaque credentials through Web Access Management.
type AuthenticationClient interface {
	resolver.UserAuthenticationService
	resolver.ProjectAuthenticationService
	resolver.OrganizationAuthenticationService
}

type authenticationClient struct {
	internalClient clients.ServiceInternalClient
	client         protos.AuthenticationServiceClient
}

// NewAuthenticationClient constructs a Web credential resolver client.
func NewAuthenticationClient(cfg *config.AppConfig, logger commons.Logger, redis connectors.RedisConnector) AuthenticationClient {
	connection, err := grpc.NewClient(cfg.Web.Host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatalf("unable to create Web authentication connection: %v", err)
	}
	return NewAuthenticationClientWithClient(cfg, logger, redis, protos.NewAuthenticationServiceClient(connection))
}

// NewAuthenticationClientWithClient constructs a resolver around an owned gRPC client.
func NewAuthenticationClientWithClient(
	cfg *config.AppConfig,
	logger commons.Logger,
	redis connectors.RedisConnector,
	client protos.AuthenticationServiceClient,
) AuthenticationClient {
	return &authenticationClient{
		internalClient: clients.NewWebInternalClient(cfg, logger, redis),
		client:         client,
	}
}

// ResolveUserCredential resolves one User session without caching credential or authorization facts.
func (client *authenticationClient) ResolveUserCredential(ctx context.Context, token string, userID uint64, selectedOrganizationID uint64, selectedProjectID uint64) (authentication.Authentication, error) {
	requestContext, err := client.internalClient.WithService(ctx)
	if err != nil {
		return authentication.Authentication{}, err
	}
	response, err := client.client.VerifyToken(requestContext, &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_User{User: &protos.VerifyTokenRequest_UserCredential{
		Token: token, UserId: userID, SelectedOrganizationId: selectedOrganizationID, SelectedProjectId: selectedProjectID,
	}}})
	if err != nil {
		return authentication.Authentication{}, err
	}
	if response == nil || response.Authentication == nil {
		return authentication.Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	return authentication.AuthenticationFromProto(response.Authentication)
}

// ResolveProjectCredential resolves one Project credential without cross-request caching.
func (client *authenticationClient) ResolveProjectCredential(ctx context.Context, token string) (authentication.Authentication, error) {
	requestContext, err := client.internalClient.WithService(ctx)
	if err != nil {
		return authentication.Authentication{}, err
	}
	response, err := client.client.VerifyToken(requestContext, &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_Project{Project: &protos.VerifyTokenRequest_ProjectCredential{Token: token}}})
	if err != nil {
		return authentication.Authentication{}, err
	}
	if response == nil || response.Authentication == nil {
		return authentication.Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	return authentication.AuthenticationFromProto(response.Authentication)
}

// ResolveOrganizationCredential resolves one Organization credential without cross-request caching.
func (client *authenticationClient) ResolveOrganizationCredential(ctx context.Context, token string) (authentication.Authentication, error) {
	requestContext, err := client.internalClient.WithService(ctx)
	if err != nil {
		return authentication.Authentication{}, err
	}
	response, err := client.client.VerifyToken(requestContext, &protos.VerifyTokenRequest{Credential: &protos.VerifyTokenRequest_Organization{Organization: &protos.VerifyTokenRequest_OrganizationCredential{Token: token}}})
	if err != nil {
		return authentication.Authentication{}, err
	}
	if response == nil || response.Authentication == nil {
		return authentication.Authentication{}, pkg_errors.AuthenticationInvalidError{}
	}
	return authentication.AuthenticationFromProto(response.Authentication)
}

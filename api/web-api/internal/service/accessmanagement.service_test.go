// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_service_test

import (
	"testing"

	internal_services "github.com/rapidaai/api/web-api/internal/service"
	"github.com/rapidaai/api/web-api/internal/service/accessmanagement"
	"github.com/stretchr/testify/require"
)

func TestNewAccessManagementServiceReturnsContract(t *testing.T) {
	var service internal_services.AccessManagementService = accessmanagement.NewAccessManagementService(nil)

	require.NotNil(t, service)
}

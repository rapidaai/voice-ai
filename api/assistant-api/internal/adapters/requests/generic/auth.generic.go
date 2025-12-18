// Copyright (c) Rapida
// Author: Prashant <prashant@rapida.ai>
//
// Licensed under the Rapida internal use license.
// This file is part of Rapida's proprietary software.
// Unauthorized copying, modification, or redistribution is strictly prohibited.
package internal_adapter_request_generic

import "github.com/rapidaai/pkg/types"

/*
 * Auth retrieves the authentication information associated with the debugger.
 *
 * This method returns the SimplePrinciple object that represents the current
 * authentication state of the debugger. The SimplePrinciple typically contains
 * information such as user ID, roles, or any other relevant authentication data.
 *
 * Returns:
 *   - types.SimplePrinciple: The authentication information for the debugger.
 */
func (dm *GenericRequestor) Auth() types.SimplePrinciple {
	return dm.auth
}

/*
 * SetAuth sets the authentication information for the debugger.
 *
 * This method allows updating the authentication state of the debugger by
 * providing a new SimplePrinciple object. This is typically used when the
 * authentication state changes, such as after a successful login or when
 * switching users.
 *
 * Parameters:
 *   - auth: types.SimplePrinciple - The new authentication information to set.
 */
func (deb *GenericRequestor) SetAuth(auth types.SimplePrinciple) {
	deb.auth = auth
}

/*
 * GetOrganizationId retrieves the current organization ID from the authentication information.
 *
 * This method returns a pointer to the uint64 representing the current organization ID
 * associated with the authenticated user. It delegates the retrieval to the auth object.
 *
 * Returns:
 *   - *uint64: A pointer to the current organization ID, or nil if not set.
 */
func (requestor *GenericRequestor) GetOrganizationId() *uint64 {
	return requestor.auth.GetCurrentOrganizationId()
}

/*
 * GetCurrentProjectId retrieves the current project ID from the authentication information.
 *
 * This method returns a pointer to the uint64 representing the current project ID
 * associated with the authenticated user. It delegates the retrieval to the auth object.
 *
 * Returns:
 *   - *uint64: A pointer to the current project ID, or nil if not set.
 */
func (requestor *GenericRequestor) GetCurrentProjectId() *uint64 {
	return requestor.auth.GetCurrentProjectId()
}

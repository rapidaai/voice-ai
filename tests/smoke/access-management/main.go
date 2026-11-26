package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/rapidaai/pkg/types"
	"github.com/rapidaai/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	organizationReadPermission = "organization.read"
	organizationRolesRead      = "organization.roles.read"
	projectRolesRead           = "project.roles.read"
	customRoleKey              = "ci_smoke_role"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connection, err := grpc.NewClient("web-api:9001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Errorf("connect to web-api: %w", err))
	}
	defer connection.Close()

	client := protos.NewAccessManagementServiceClient(connection)
	projectCtx := projectContext(ctx)
	verifyProjectReads(projectCtx, client)
	verifyProjectMutationDenials(projectCtx, client)
	verifyUserLifecycle(userContext(ctx), client)
	fmt.Println("All access management API smoke tests passed")
}

func verifyProjectMutationDenials(ctx context.Context, client protos.AccessManagementServiceClient) {
	_, err := client.CreateCustomRole(requestContext(ctx, "project-create-role-denied"), &protos.CreateCustomRoleRequest{})
	requirePermissionDenied("create custom role with project API key", err)

	_, err = client.UpdateCustomRole(requestContext(ctx, "project-update-role-denied"), &protos.UpdateCustomRoleRequest{})
	requirePermissionDenied("update custom role with project API key", err)

	_, err = client.ReduceCustomRolePermissions(requestContext(ctx, "project-reduce-role-denied"), &protos.ReduceCustomRolePermissionsRequest{})
	requirePermissionDenied("reduce custom role with project API key", err)

	_, err = client.ArchiveCustomRole(requestContext(ctx, "project-archive-role-denied"), &protos.ArchiveCustomRoleRequest{})
	requirePermissionDenied("archive custom role with project API key", err)

	fmt.Println("Access management mutation APIs rejected project API key authentication")
}

func userContext(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		types.AUTHORIZATION_KEY, requiredEnvironment("CI_STACK_AUTH_TOKEN"),
		types.AUTH_KEY, "1",
		types.PROJECT_KEY, "1",
	))
}

func projectContext(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(
		types.PROJECT_SCOPE_KEY, types.PROJECT_KEY_PREFIX+requiredEnvironment("CI_STACK_PROJECT_API_KEY"),
	))
}

func requestContext(ctx context.Context, operation string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, types.REQUEST_ID_KEY, "ci-access-management-"+operation)
}

func requiredEnvironment(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Errorf("%s is required", key))
	}
	return value
}

func verifyProjectReads(ctx context.Context, client protos.AccessManagementServiceClient) {
	permissions, err := client.ListPermissions(requestContext(ctx, "project-list-permissions"), &protos.ListPermissionsRequest{
		Paginate:  &protos.Paginate{Page: 1, PageSize: 100},
		Criterias: []*protos.Criteria{{Key: "scope", Logic: "=", Value: "PROJECT"}},
	})
	requireResponse("list project permissions", permissions.GetSuccess(), permissions.GetCode(), permissions.GetError(), err)
	projectPermission := permissionByKey(permissions.GetPermissions(), projectRolesRead)
	if projectPermission == nil {
		panic(fmt.Errorf("list project permissions omitted %q", projectRolesRead))
	}

	roles, err := client.ListRoles(requestContext(ctx, "project-list-roles"), &protos.ListRolesRequest{
		Paginate: &protos.Paginate{Page: 1, PageSize: 100},
		Criterias: []*protos.Criteria{
			{Key: "scope", Logic: "=", Value: "PROJECT"},
			{Key: "subject", Logic: "=", Value: "PROJECT_TOKEN"},
		},
	})
	requireResponse("list project token roles", roles.GetSuccess(), roles.GetCode(), roles.GetError(), err)
	if len(roles.GetRoles()) == 0 {
		panic("list project token roles returned no roles")
	}

	role, err := client.GetRole(requestContext(ctx, "project-get-role"), &protos.GetRoleRequest{RoleDefinitionId: roles.GetRoles()[0].GetId()})
	requireResponse("get project token role", role.GetSuccess(), role.GetCode(), role.GetError(), err)
	if role.GetRole().GetId() != roles.GetRoles()[0].GetId() {
		panic(fmt.Errorf("get project token role returned id %d; expected %d", role.GetRole().GetId(), roles.GetRoles()[0].GetId()))
	}
	fmt.Println("Access management read APIs passed with project API key authentication")
}

func verifyUserLifecycle(ctx context.Context, client protos.AccessManagementServiceClient) {
	permissions, err := client.ListPermissions(requestContext(ctx, "user-list-permissions"), &protos.ListPermissionsRequest{
		Paginate:  &protos.Paginate{Page: 1, PageSize: 100},
		Criterias: []*protos.Criteria{{Key: "scope", Logic: "=", Value: "ORGANIZATION"}},
	})
	requireResponse("list organization permissions", permissions.GetSuccess(), permissions.GetCode(), permissions.GetError(), err)
	readPermission := permissionByKey(permissions.GetPermissions(), organizationReadPermission)
	rolesReadPermission := permissionByKey(permissions.GetPermissions(), organizationRolesRead)
	if readPermission == nil || rolesReadPermission == nil {
		panic(fmt.Errorf("list organization permissions omitted required permissions %q or %q", organizationReadPermission, organizationRolesRead))
	}

	created, err := client.CreateCustomRole(requestContext(ctx, "user-create-role"), &protos.CreateCustomRoleRequest{
		Key: customRoleKey, Name: "CI Smoke Role", Description: "Created by the access management smoke test.",
		Scope: protos.RoleScope_ROLE_SCOPE_ORGANIZATION, Subject: protos.RoleSubject_ROLE_SUBJECT_USER,
		PermissionDefinitionIds: []uint64{readPermission.GetId(), rolesReadPermission.GetId()},
	})
	requireResponse("create custom role", created.GetSuccess(), created.GetCode(), created.GetError(), err)
	roleID := created.GetRole().GetId()
	if roleID == 0 || created.GetRole().GetKey() != customRoleKey {
		panic(fmt.Errorf("create custom role returned id=%d key=%q", roleID, created.GetRole().GetKey()))
	}

	roles, err := client.ListRoles(requestContext(ctx, "user-list-roles"), &protos.ListRolesRequest{
		Paginate:  &protos.Paginate{Page: 1, PageSize: 100},
		Criterias: []*protos.Criteria{{Key: "key", Logic: "contains", Value: customRoleKey}},
	})
	requireResponse("list custom role", roles.GetSuccess(), roles.GetCode(), roles.GetError(), err)
	if len(roles.GetRoles()) != 1 || roles.GetRoles()[0].GetId() != roleID {
		panic(fmt.Errorf("list custom role returned %d roles for id %d", len(roles.GetRoles()), roleID))
	}

	updated, err := client.UpdateCustomRole(requestContext(ctx, "user-update-role"), &protos.UpdateCustomRoleRequest{
		RoleDefinitionId: roleID, Name: "CI Smoke Role Updated", Description: "Updated by the access management smoke test.",
	})
	requireResponse("update custom role", updated.GetSuccess(), updated.GetCode(), updated.GetError(), err)
	if updated.GetRole().GetName() != "CI Smoke Role Updated" {
		panic(fmt.Errorf("update custom role returned name %q", updated.GetRole().GetName()))
	}

	role, err := client.GetRole(requestContext(ctx, "user-get-role"), &protos.GetRoleRequest{RoleDefinitionId: roleID})
	requireResponse("get custom role", role.GetSuccess(), role.GetCode(), role.GetError(), err)
	if role.GetRole().GetId() != roleID || len(role.GetRole().GetPermissions()) != 2 {
		panic(fmt.Errorf("get custom role returned id=%d permissions=%d", role.GetRole().GetId(), len(role.GetRole().GetPermissions())))
	}

	preview, err := client.ReduceCustomRolePermissions(requestContext(ctx, "user-preview-reduction"), &protos.ReduceCustomRolePermissionsRequest{
		RoleDefinitionId: roleID, PermissionDefinitionIds: []uint64{rolesReadPermission.GetId()},
		Reason: "Verify role permission reduction in smoke coverage.", Preview: true,
	})
	requireResponse("preview custom role reduction", preview.GetSuccess(), preview.GetCode(), preview.GetError(), err)
	digest := preview.GetPreview().GetConfirmationDigest()
	if len(digest) != 64 || preview.GetCorrelationId() == "" {
		panic(fmt.Errorf("preview custom role reduction returned digest length=%d correlation=%q", len(digest), preview.GetCorrelationId()))
	}
	if got := permissionKeys(preview.GetPreview().GetResultingRole().GetPermissions()); !slices.Equal(got, []string{organizationReadPermission}) {
		panic(fmt.Errorf("preview custom role reduction returned permissions %v", got))
	}

	reduced, err := client.ReduceCustomRolePermissions(requestContext(ctx, "user-apply-reduction"), &protos.ReduceCustomRolePermissionsRequest{
		RoleDefinitionId: roleID, PermissionDefinitionIds: []uint64{rolesReadPermission.GetId()},
		Reason: "Verify role permission reduction in smoke coverage.", ConfirmationDigest: digest,
	})
	requireResponse("apply custom role reduction", reduced.GetSuccess(), reduced.GetCode(), reduced.GetError(), err)
	if got := permissionKeys(reduced.GetRole().GetPermissions()); !slices.Equal(got, []string{organizationReadPermission}) {
		panic(fmt.Errorf("apply custom role reduction returned permissions %v", got))
	}

	archived, err := client.ArchiveCustomRole(requestContext(ctx, "user-archive-role"), &protos.ArchiveCustomRoleRequest{
		RoleDefinitionId: roleID, Reason: "Complete access management smoke lifecycle.",
	})
	requireResponse("archive custom role", archived.GetSuccess(), archived.GetCode(), archived.GetError(), err)
	if archived.GetRoleDefinitionId() != roleID || archived.GetCorrelationId() == "" {
		panic(fmt.Errorf("archive custom role returned id=%d correlation=%q", archived.GetRoleDefinitionId(), archived.GetCorrelationId()))
	}
	fmt.Println("All access management APIs passed with personal access token authentication")
}

func permissionByKey(permissions []*protos.PermissionDefinition, key string) *protos.PermissionDefinition {
	for _, permission := range permissions {
		if permission.GetKey() == key {
			return permission
		}
	}
	return nil
}

func permissionKeys(permissions []*protos.PermissionDefinition) []string {
	keys := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		keys = append(keys, permission.GetKey())
	}
	return keys
}

func requireResponse(operation string, success bool, code int32, responseError *protos.Error, callError error) {
	if callError != nil {
		panic(fmt.Errorf("%s: %w", operation, callError))
	}
	if !success || code != 200 {
		panic(fmt.Errorf("%s returned success=%t code=%d error=%v", operation, success, code, responseError))
	}
}

func requirePermissionDenied(operation string, err error) {
	if status.Code(err) != codes.PermissionDenied {
		panic(fmt.Errorf("%s returned %v; expected permission denied", operation, err))
	}
}

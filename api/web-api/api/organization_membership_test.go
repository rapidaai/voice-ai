//go:build integration && cgo
// +build integration,cgo

package web_api

import (
	"context"
	"errors"
	"testing"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
)

func TestGetAllUserReturnsActiveAndInvitedOrganizationMembers(t *testing.T) {
	projectApi, db, _ := newProjectAPITest(t)
	api := &webAuthGRPCApi{
		webAuthApi: webAuthApi{
			logger:      projectApi.logger,
			postgres:    projectApi.postgres,
			userService: projectApi.userService,
		},
	}
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 80},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Active",
		Email:    "active@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 81},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_INVITED,
		},
		Name:     "Invited",
		Email:    "invited@example.com",
		Password: "hash",
		Source:   "invited-by-other",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 82},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Other",
		Email:    "other@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 83},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     80,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 84},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_INVITED,
		},
		UserAuthId:     81,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 85},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     82,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.GetAllUser(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.GetAllUserRequest{
		Paginate: &protos.Paginate{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.EqualValues(t, 2, res.GetPaginated().GetTotalItem())

	members := map[string]string{}
	for _, member := range res.GetData() {
		members[member.GetEmail()] = member.GetStatus()
	}
	require.Equal(t, map[string]string{
		"active@example.com":  type_enums.RECORD_ACTIVE.String(),
		"invited@example.com": type_enums.RECORD_INVITED.String(),
	}, members)
}

func TestInviteUserToOrganizationRejectsAuthAndValidationFailures(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)

	res, err := api.InviteUserToOrganization(context.Background(), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.Error(t, err)
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.ErrorMessage, err.Error())
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUnauthenticated.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.ErrorMessage, res.GetError().GetHumanMessage())

	tests := []struct {
		name          string
		ctx           context.Context
		req           *protos.InviteUserToOrganizationRequest
		platformError pkg_errors.PlatformError
		expectError   bool
	}{
		{
			name:          "non admin",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_MEMBER.String()),
			platformError: pkg_errors.InviteUserToOrganizationUnauthorized,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "invalid raw email",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidEmail,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            " new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "invalid organization role",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: "Member",
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "owner invite role",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new-owner@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_OWNER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "invalid project role",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidProjectRole,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: "Reader"},
				},
			},
		},
		{
			name:          "empty project id",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidProjects,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 0, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "duplicate project",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationDuplicateProject,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_WRITER.String()},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.InviteUserToOrganization(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
				require.Equal(t, tt.platformError.ErrorMessage, err.Error())
			} else {
				require.NoError(t, err)
			}
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
			var count int64
			require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestInviteUserToOrganizationRejectsInvalidProjectsBeforeWrites(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)

	for _, ids := range [][]uint64{{200}, {300}, {999}, {100, 200}} {
		projectRoles := []*protos.ProjectRoleAssignment{}
		for _, id := range ids {
			projectRoles = append(projectRoles, &protos.ProjectRoleAssignment{
				ProjectId:   id,
				ProjectRole: type_enums.PROJECT_ROLE_WRITER.String(),
			})
		}
		res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.InviteUserToOrganizationRequest{
			Email:            "new@example.com",
			OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
			ProjectRoles:     projectRoles,
		})
		require.NoError(t, err)
		require.False(t, res.GetSuccess())
		require.Equal(t, pkg_errors.InviteUserToOrganizationInvalidProjects.HTTPStatusCodeInt32(), res.GetCode())
		require.EqualValues(t, pkg_errors.InviteUserToOrganizationInvalidProjects.Code, res.GetError().GetErrorCode())
		require.Equal(t, pkg_errors.InviteUserToOrganizationInvalidProjects.Error, res.GetError().GetErrorMessage())
		require.Equal(t, pkg_errors.InviteUserToOrganizationInvalidProjects.ErrorMessage, res.GetError().GetHumanMessage())
		var count int64
		require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
		require.Zero(t, count)
	}
}

func TestInviteUserToOrganizationInvitesNewUserWithOrganizationAndProjectRoles(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_WRITER.String()},
			{ProjectId: 101, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, "new@example.com", res.GetData().GetEmail())
	require.Equal(t, type_enums.ORGANIZATION_ROLE_ADMIN.String(), res.GetData().GetRole())
	require.Equal(t, 1, emailer.calls)
	require.Equal(t, "new@example.com", emailer.to.Email)
	require.Equal(t, "Alpha,Beta", emailer.args["project_name"])

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "email = ?", "new@example.com").Error)
	var orgRole internal_entity.UserOrganizationRole
	require.NoError(t, db.First(&orgRole, "user_auth_id = ?", user.Id).Error)
	require.Equal(t, type_enums.ORGANIZATION_ROLE_ADMIN.String(), orgRole.Role)
	require.Equal(t, type_enums.RECORD_INVITED, orgRole.Status)
	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ?", user.Id).Error)
	require.Len(t, projectRoles, 2)
	roles := map[uint64]string{}
	for _, projectRole := range projectRoles {
		roles[projectRole.ProjectId] = projectRole.Role
		require.Equal(t, type_enums.RECORD_INVITED, projectRole.Status)
	}
	require.Equal(t, map[uint64]string{
		100: type_enums.PROJECT_ROLE_WRITER.String(),
		101: type_enums.PROJECT_ROLE_READER.String(),
	}, roles)
}

func TestInviteUserToOrganizationRejectsExistingSameOrgUser(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 50},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Existing",
		Email:    "existing@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 51},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     50,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 52},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId: 50,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "existing@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_ADMIN.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.ErrorMessage, res.GetError().GetHumanMessage())

	var orgRoles []internal_entity.UserOrganizationRole
	require.NoError(t, db.Find(&orgRoles, "user_auth_id = ? AND organization_id = ?", 50, 10).Error)
	require.Len(t, orgRoles, 1)
	require.Equal(t, type_enums.ORGANIZATION_ROLE_MEMBER.String(), orgRoles[0].Role)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ? AND project_id = ?", 50, 100).Error)
	require.Len(t, projectRoles, 1)
	require.Equal(t, type_enums.PROJECT_ROLE_READER.String(), projectRoles[0].Role)
	require.Equal(t, type_enums.RECORD_ACTIVE, projectRoles[0].Status)
}

func TestInviteUserToOrganizationRejectsExistingInvitedSameOrgUser(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 60},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_INVITED,
		},
		Name:     "Invited",
		Email:    "invited@example.com",
		Password: "hash",
		Source:   "invited-by-other",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 61},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_INVITED,
		},
		UserAuthId:     60,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "invited@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.ErrorMessage, res.GetError().GetHumanMessage())

	var orgRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserOrganizationRole{}).Where("user_auth_id = ?", 60).Count(&orgRoleCount).Error)
	require.EqualValues(t, 1, orgRoleCount)

	var projectRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ?", 60).Count(&projectRoleCount).Error)
	require.Zero(t, projectRoleCount)
}

func TestInviteUserToOrganizationRejectsExistingInvitedUserInAnotherOrganization(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 70},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_INVITED,
		},
		Name:     "Invited",
		Email:    "cross-invited@example.com",
		Password: "hash",
		Source:   "invited-by-other",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 71},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_INVITED,
		},
		UserAuthId:     70,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "cross-invited@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.ErrorMessage, res.GetError().GetHumanMessage())
	require.Zero(t, emailer.calls)

	var projectRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ?", 70).Count(&projectRoleCount).Error)
	require.Zero(t, projectRoleCount)
}

func TestInviteUserToOrganizationRestoresArchivedUserAsActive(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 80},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ARCHIEVE,
		},
		Name:     "Archived",
		Email:    "archived@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 81},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ARCHIEVE,
		},
		UserAuthId:     80,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 82},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ARCHIEVE,
		},
		UserAuthId: 80,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "archived@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_WRITER.String()},
			{ProjectId: 101, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, type_enums.RECORD_ACTIVE.String(), res.GetData().GetStatus())
	require.Equal(t, 1, emailer.calls)
	require.Contains(t, emailer.args["invite_url"], "/auth/signin")

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "id = ?", 80).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, user.Status)

	var orgRoles []internal_entity.UserOrganizationRole
	require.NoError(t, db.Order("id ASC").Find(&orgRoles, "user_auth_id = ? AND organization_id = ?", 80, 10).Error)
	require.Len(t, orgRoles, 2)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, orgRoles[0].Status)
	require.Equal(t, type_enums.RECORD_ACTIVE, orgRoles[1].Status)
	require.Equal(t, type_enums.ORGANIZATION_ROLE_ADMIN.String(), orgRoles[1].Role)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ? AND status = ?", 80, type_enums.RECORD_ACTIVE.String()).Error)
	require.Len(t, projectRoles, 2)
	activeRoles := map[uint64]string{}
	for _, projectRole := range projectRoles {
		activeRoles[projectRole.ProjectId] = projectRole.Role
	}
	require.Equal(t, map[uint64]string{
		100: type_enums.PROJECT_ROLE_WRITER.String(),
		101: type_enums.PROJECT_ROLE_READER.String(),
	}, activeRoles)
}

func TestInviteUserToOrganizationProjectRoleFailureReturnsError(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Exec(`DROP TABLE user_project_roles`).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.ErrorMessage, res.GetError().GetHumanMessage())

	var count int64
	require.NoError(t, db.Model(&internal_entity.UserAuth{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestInviteUserToOrganizationEmailFailureDoesNotRollback(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)
	emailer.err = errors.New("email failed")

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, 1, emailer.calls)

	var count int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestUpdateUserOrganizationRoleUpdatesActiveOrganizationMember(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 100},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Role Update",
		Email:    "role-update@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 101},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     100,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.UpdateUserOrganizationRole(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.UpdateUserOrganizationRoleRequest{
		UserId:           100,
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.EqualValues(t, 200, res.GetCode())

	var orgRole internal_entity.UserOrganizationRole
	require.NoError(t, db.First(&orgRole, "user_auth_id = ? AND organization_id = ?", 100, 10).Error)
	require.Equal(t, type_enums.ORGANIZATION_ROLE_ADMIN.String(), orgRole.Role)
	require.Equal(t, type_enums.RECORD_ACTIVE, orgRole.Status)
}

func TestUpdateUserOrganizationRoleRejectsAuthAndValidationFailures(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 102},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Outside Org",
		Email:    "outside-org@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 103},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     102,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 104},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Owner Target",
		Email:    "update-owner@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 105},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     104,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_OWNER.String(),
	}).Error)

	res, err := api.UpdateUserOrganizationRole(context.Background(), &protos.UpdateUserOrganizationRoleRequest{
		UserId:           102,
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
	})
	require.Error(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.UpdateUserOrganizationRoleUnauthenticated.HTTPStatusCodeInt32(), res.GetCode())

	tests := []struct {
		name          string
		ctx           context.Context
		req           *protos.UpdateUserOrganizationRoleRequest
		platformError pkg_errors.PlatformError
		expectError   bool
	}{
		{
			name:          "non admin",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_MEMBER.String()),
			req:           &protos.UpdateUserOrganizationRoleRequest{UserId: 102, OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String()},
			platformError: pkg_errors.UpdateUserOrganizationRoleUnauthorized,
		},
		{
			name:          "zero user id",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.UpdateUserOrganizationRoleRequest{OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String()},
			platformError: pkg_errors.UpdateUserOrganizationRoleInvalidUser,
		},
		{
			name:          "invalid organization role",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.UpdateUserOrganizationRoleRequest{UserId: 102, OrganizationRole: "Admin"},
			platformError: pkg_errors.UpdateUserOrganizationRoleInvalidRole,
		},
		{
			name:          "owner role request",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.UpdateUserOrganizationRoleRequest{UserId: 102, OrganizationRole: type_enums.ORGANIZATION_ROLE_OWNER.String()},
			platformError: pkg_errors.UpdateUserOrganizationRoleInvalidRole,
		},
		{
			name:          "user outside organization",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.UpdateUserOrganizationRoleRequest{UserId: 102, OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String()},
			platformError: pkg_errors.UpdateUserOrganizationRoleUserNotInOrg,
		},
		{
			name:          "organization owner",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()),
			req:           &protos.UpdateUserOrganizationRoleRequest{UserId: 104, OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String()},
			platformError: pkg_errors.UpdateUserOrganizationRoleOwner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.UpdateUserOrganizationRole(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
		})
	}
}

func TestDeleteUserFromOrganizationArchivesUserOrganizationAndProjectRoles(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 110},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Delete Org",
		Email:    "delete-org@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 111},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     110,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 112},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId: 110,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 113},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_INVITED,
		},
		UserAuthId: 110,
		ProjectId:  101,
		Role:       type_enums.PROJECT_ROLE_WRITER.String(),
	}).Error)

	res, err := api.DeleteUserFromOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.DeleteUserFromOrganizationRequest{
		UserId: 110,
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.EqualValues(t, 110, res.GetId())

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "id = ?", 110).Error)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, user.Status)

	var orgRole internal_entity.UserOrganizationRole
	require.NoError(t, db.First(&orgRole, "user_auth_id = ? AND organization_id = ?", 110, 10).Error)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, orgRole.Status)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ?", 110).Error)
	require.Len(t, projectRoles, 2)
	for _, projectRole := range projectRoles {
		require.Equal(t, type_enums.RECORD_ARCHIEVE, projectRole.Status)
	}
}

func TestDeleteUserFromOrganizationRejectsAuthAndValidationFailures(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 120},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Outside",
		Email:    "delete-outside@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 121},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     120,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 122},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		Name:     "Owner Target",
		Email:    "delete-owner@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 123},
		Mutable: gorm_models.Mutable{
			Status: type_enums.RECORD_ACTIVE,
		},
		UserAuthId:     122,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_OWNER.String(),
	}).Error)

	res, err := api.DeleteUserFromOrganization(context.Background(), &protos.DeleteUserFromOrganizationRequest{UserId: 120})
	require.Error(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.DeleteUserFromOrganizationUnauthenticated.HTTPStatusCodeInt32(), res.GetCode())

	tests := []struct {
		name          string
		ctx           context.Context
		req           *protos.DeleteUserFromOrganizationRequest
		platformError pkg_errors.PlatformError
		expectError   bool
	}{
		{
			name:          "non admin",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_MEMBER.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{UserId: 120},
			platformError: pkg_errors.DeleteUserFromOrganizationUnauthorized,
		},
		{
			name:          "zero user id",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{},
			platformError: pkg_errors.DeleteUserFromOrganizationInvalidUser,
		},
		{
			name:          "user outside organization",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{UserId: 120},
			platformError: pkg_errors.DeleteUserFromOrganizationUserNotInOrg,
		},
		{
			name:          "organization owner",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{UserId: 122},
			platformError: pkg_errors.DeleteUserFromOrganizationOwner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.DeleteUserFromOrganization(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
		})
	}
}

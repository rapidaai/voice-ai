// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>

// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.

package internal_user_service

import (
	"context"
	"strings"
	"testing"
	"time"

	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
)

func TestGetAllOrganizationMemberPaginatesUsersWithCurrentOrganizationRoles(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	memberCreated := time.Date(2026, time.January, 3, 0, 0, 0, 0, time.UTC)
	invitedCreated := memberCreated.Add(-24 * time.Hour)
	ownerCreated := invitedCreated.Add(-24 * time.Hour)
	if err := db.Exec("UPDATE user_auths SET created_date = ? WHERE id = ?", ownerCreated, 1).Error; err != nil {
		t.Fatal(err)
	}
	users := []internal_entity.UserAuth{
		{
			Audited: gorm_models.Audited{Id: 2, CreatedDate: gorm_models.TimeWrapper(memberCreated)},
			Mutable: gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			Name:    "Member", Email: "member@example.com", Password: "hash", Source: "direct",
		},
		{
			Audited: gorm_models.Audited{Id: 3, CreatedDate: gorm_models.TimeWrapper(memberCreated.Add(time.Hour))},
			Mutable: gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			Name:    "Foreign", Email: "foreign@example.com", Password: "hash", Source: "direct",
		},
		{
			Audited: gorm_models.Audited{Id: 4, CreatedDate: gorm_models.TimeWrapper(memberCreated.Add(2 * time.Hour))},
			Mutable: gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			Name:    "Roleless", Email: "roleless@example.com", Password: "hash", Source: "direct",
		},
		{
			Audited: gorm_models.Audited{Id: 5, CreatedDate: gorm_models.TimeWrapper(invitedCreated)},
			Mutable: gorm_models.Mutable{Status: type_enums.RECORD_INVITED},
			Name:    "Invited", Email: "invited@example.com", Password: "hash", Source: "invited-by-other",
		},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("UPDATE user_organization_roles SET created_date = ? WHERE user_auth_id = ?", memberCreated.Add(72*time.Hour), 1).Error; err != nil {
		t.Fatal(err)
	}
	assignments := []internal_entity.UserOrganizationRole{
		{
			Audited:    gorm_models.Audited{CreatedDate: gorm_models.TimeWrapper(memberCreated)},
			Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			UserAuthId: 2, OrganizationId: 10, RoleDefinitionID: 102,
			RoleScopeType: "ORGANIZATION", RoleSubjectType: "USER",
		},
		{
			Audited:    gorm_models.Audited{CreatedDate: gorm_models.TimeWrapper(memberCreated)},
			Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			UserAuthId: 2, OrganizationId: 10, RoleDefinitionID: 108,
			RoleScopeType: "ORGANIZATION", RoleSubjectType: "USER",
		},
		{
			Audited:    gorm_models.Audited{CreatedDate: gorm_models.TimeWrapper(memberCreated.Add(48 * time.Hour))},
			Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			UserAuthId: 3, OrganizationId: 20, RoleDefinitionID: 108,
			RoleScopeType: "ORGANIZATION", RoleSubjectType: "USER",
		},
		{
			Audited:    gorm_models.Audited{CreatedDate: gorm_models.TimeWrapper(memberCreated.Add(24 * time.Hour))},
			Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_INVITED},
			UserAuthId: 5, OrganizationId: 10, RoleDefinitionID: 108,
			RoleScopeType: "ORGANIZATION", RoleSubjectType: "USER",
		},
	}
	if err := db.Create(&assignments).Error; err != nil {
		t.Fatal(err)
	}

	count, roles, err := service.GetAllOrganizationMember(
		context.Background(),
		10,
		nil,
		&protos.Paginate{Page: 1, PageSize: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
	wantUsers := []uint64{2, 2, 5}
	if len(*roles) != len(wantUsers) {
		t.Fatalf("roles = %d, want %d", len(*roles), len(wantUsers))
	}
	for index, role := range *roles {
		if role.UserAuthId != wantUsers[index] {
			t.Fatalf("role[%d].UserAuthId = %d, want %d", index, role.UserAuthId, wantUsers[index])
		}
		if role.Member.Id != role.UserAuthId {
			t.Fatalf("role[%d].Member.Id = %d, want %d", index, role.Member.Id, role.UserAuthId)
		}
		if role.RoleDefinition.Id != role.RoleDefinitionID {
			t.Fatalf("role[%d].RoleDefinition.Id = %d, want %d", index, role.RoleDefinition.Id, role.RoleDefinitionID)
		}
	}

	count, roles, err = service.GetAllOrganizationMember(
		context.Background(),
		10,
		nil,
		&protos.Paginate{Page: 2, PageSize: 2},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 || len(*roles) != 1 || (*roles)[0].UserAuthId != 1 {
		t.Fatalf("second page count = %d, roles = %+v, want owner with total 3", count, *roles)
	}
}

func TestGetAllOrganizationMemberAppliesCriteriaToUsers(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	if err := db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 2},
		Mutable: gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
		Name:    "Member", Email: "member@example.com", Password: "hash", Source: "direct",
	}).Error; err != nil {
		t.Fatal(err)
	}
	for _, roleDefinitionID := range []uint64{102, 108} {
		if err := db.Create(&internal_entity.UserOrganizationRole{
			Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			UserAuthId: 2, OrganizationId: 10, RoleDefinitionID: roleDefinitionID,
			RoleScopeType: "ORGANIZATION", RoleSubjectType: "USER",
		}).Error; err != nil {
			t.Fatal(err)
		}
	}

	count, roles, err := service.GetAllOrganizationMember(
		context.Background(),
		10,
		[]*protos.Criteria{{Key: "email", Value: "member@example.com"}},
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(*roles) != 2 {
		t.Fatalf("count = %d, roles = %d, want 1 user and 2 roles", count, len(*roles))
	}
	for _, role := range *roles {
		if role.UserAuthId != 2 {
			t.Fatalf("role.UserAuthId = %d, want 2", role.UserAuthId)
		}
	}
}

func TestGetAllOrganizationMemberAppliesStatusCriteriaToOrganizationRole(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	if err := db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 2},
		Mutable: gorm_models.Mutable{Status: type_enums.RECORD_INVITED},
		Name:    "Member", Email: "member@example.com", Password: "hash", Source: "direct",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&internal_entity.UserOrganizationRole{
		Mutable:    gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
		UserAuthId: 2, OrganizationId: 10, RoleDefinitionID: 102,
		RoleScopeType: "ORGANIZATION", RoleSubjectType: "USER",
	}).Error; err != nil {
		t.Fatal(err)
	}

	count, roles, err := service.GetAllOrganizationMember(
		context.Background(),
		10,
		[]*protos.Criteria{{Key: "status", Value: type_enums.RECORD_ACTIVE.String()}},
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || len(*roles) != 2 {
		t.Fatalf("active count = %d, roles = %d, want owner and member", count, len(*roles))
	}

	count, roles, err = service.GetAllOrganizationMember(
		context.Background(),
		10,
		[]*protos.Criteria{{Key: "status", Value: type_enums.RECORD_INVITED.String()}},
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 || len(*roles) != 0 {
		t.Fatalf("invited count = %d, roles = %d, want no organization roles", count, len(*roles))
	}
}

func TestGetAllOrganizationMemberAppliesContainsCriteriaLiterally(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	users := []internal_entity.UserAuth{
		{
			Audited: gorm_models.Audited{Id: 2}, Mutable: gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			Name: "Percent", Email: "p_iyer%25@example.com", Password: "hash", Source: "direct",
		},
		{
			Audited: gorm_models.Audited{Id: 3}, Mutable: gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE},
			Name: "Other", Email: "plain@example.com", Password: "hash", Source: "direct",
		},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatal(err)
	}
	for _, userID := range []uint64{2, 3} {
		if err := db.Create(&internal_entity.UserOrganizationRole{
			Mutable: gorm_models.Mutable{Status: type_enums.RECORD_ACTIVE}, UserAuthId: userID,
			OrganizationId: 10, RoleDefinitionID: 102, RoleScopeType: "ORGANIZATION", RoleSubjectType: "USER",
		}).Error; err != nil {
			t.Fatal(err)
		}
	}

	count, roles, err := service.GetAllOrganizationMember(
		context.Background(),
		10,
		[]*protos.Criteria{{Key: "email", Logic: "contains", Value: "p_iyer%"}},
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(*roles) != 1 || (*roles)[0].UserAuthId != 2 {
		t.Fatalf("count = %d, roles = %+v, want literal partial match for user 2", count, *roles)
	}
}

func TestGetAllOrganizationMemberRejectsUnsupportedCriteria(t *testing.T) {
	service, _, _ := newAssignmentServiceTest(t)

	_, _, err := service.GetAllOrganizationMember(
		context.Background(),
		10,
		[]*protos.Criteria{{Key: "unknown", Value: "value"}},
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported organization member criterion") {
		t.Fatalf("GetAllOrganizationMember() error = %v, want unsupported criterion error", err)
	}
}

func TestGetAllOrganizationMemberRejectsNilCriteria(t *testing.T) {
	service, _, _ := newAssignmentServiceTest(t)

	_, _, err := service.GetAllOrganizationMember(
		context.Background(),
		10,
		[]*protos.Criteria{nil},
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err == nil || !strings.Contains(err.Error(), "organization member criterion is required") {
		t.Fatalf("GetAllOrganizationMember() error = %v, want required criterion error", err)
	}
}

func TestGetAllOrganizationMemberRejectsUnsupportedLogic(t *testing.T) {
	service, _, _ := newAssignmentServiceTest(t)

	_, _, err := service.GetAllOrganizationMember(
		context.Background(),
		10,
		[]*protos.Criteria{{Key: "email", Logic: "starts_with", Value: "member"}},
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported logic") {
		t.Fatalf("GetAllOrganizationMember() error = %v, want unsupported logic error", err)
	}
}

func TestGetAllOrganizationMemberReturnsPersistenceFailure(t *testing.T) {
	service, db, _ := newAssignmentServiceTest(t)
	sqlDatabase, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDatabase.Close(); err != nil {
		t.Fatal(err)
	}

	_, _, err = service.GetAllOrganizationMember(
		context.Background(),
		10,
		nil,
		&protos.Paginate{Page: 1, PageSize: 20},
	)
	if err == nil {
		t.Fatal("GetAllOrganizationMember() error = nil, want persistence failure")
	}
}

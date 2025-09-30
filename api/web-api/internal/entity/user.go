package internal_entity

import (
	"time"

	gorm_model "github.com/lexatic/web-backend/pkg/models/gorm"
)

type UserAuth struct {
	gorm_model.Audited
	gorm_model.Mutable
	Name     string `json:"name" gorm:"type:string;size:200;not null"`
	Email    string `json:"email" gorm:"type:string;size:200;not null;index:ua_idx_email"`
	Password string `json:"password" gorm:"type:string;size:200;not null"`
	// Status    string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	Source string `json:"source" gorm:"type:string;size:50;not null;default:direct"`
	// CreatedBy uint64 `json:"created_by" gorm:"type:bigint;size:20;not null"`
	// UpdatedBy uint64 `json:"updated_by" gorm:"type:bigint;size:20;"`
}

type UserSocial struct {
	gorm_model.Audited
	UserAuthId uint64 `json:"user_auth_id" gorm:"type:bigint;size:20;not null;index:up_idx_auth_id"`
	Social     string `json:"social" gorm:"type:string;size:200;not null"`
	Identifier string `json:"identifier" gorm:"type:string;size:200;not null"`
	Verified   bool   `json:"verified" gorm:"type:boolean;default:false"`
	Token      string `json:"token" gorm:"type:string;size:500;not null"`
}

// make sure you have only smaller case email in the database

func (ua *UserAuth) GetId() uint64 {
	return ua.Id
}

func (ua *UserAuth) GetName() string {
	return ua.Name
}

func (ua *UserAuth) GetEmail() string {
	return ua.Email
}

type UserFeaturePermission struct {
	gorm_model.Audited
	gorm_model.Mutable
	UserAuthId uint64 `json:"user_auth_id" gorm:"type:bigint;size:20;not null;"`
	// deployments.endpoints
	// deployments.workflows
	// deployments.assistants
	//
	// knowledges
	Feature   string `json:"feature" gorm:"type:string;size:200;not null"`
	IsEnabled bool   `json:"is_enabled" gorm:"type:bool;not null"`
	// CreatedBy uint64 `json:"created_by" gorm:"type:bigint;size:20;not null"`
	// UpdatedBy uint64 `json:"updated_by" gorm:"type:bigint;size:20;"`
	// Status    string `json:"status" gorm:"type:string;size:50;not null;default:active"`
}

type UserAuthToken struct {
	gorm_model.Audited
	gorm_model.Mutable
	UserAuthId uint64    `json:"user_auth_id" gorm:"type:bigint;size:20;not null;index:up_idx_auth_id"`
	TokenType  string    `json:"token_type" gorm:"type:string;size:100;not null;"`
	Token      string    `json:"token" gorm:"type:string;size:200;not null"`
	ExpireAt   time.Time `json:"expire_at" gorm:"type:timestamp;not null;<-:create"`
	// CreatedBy  uint64    `json:"created_by" gorm:"type:bigint;size:20;not null"`
	// UpdatedBy  uint64    `json:"updated_by" gorm:"type:bigint;size:20;"`
	// Status     string    `json:"status" gorm:"type:string;size:50;not null;default:active"`
}

func (uat *UserAuthToken) GetId() uint64 {
	return uat.Id
}

func (uat *UserAuthToken) GetToken() string {
	return uat.Token
}

func (uat *UserAuthToken) GetTokenType() string {
	return uat.TokenType
}

func (uat *UserAuthToken) IsExpired() bool {
	return false
}

type UserOrganizationRole struct {
	gorm_model.Audited
	gorm_model.Mutable
	UserAuthId     uint64 `json:"user_auth_id" gorm:"type:bigint;size:20;not null;index:ur_idx_auth_id"`
	OrganizationId uint64 `json:"organization_id" gorm:"type:bigint;size:20;not null"`
	Role           string `json:"role" gorm:"type:string;size:200;not null;"`
	// CreatedBy      uint64       `json:"created_by" gorm:"type:bigint;size:20;not null"`
	// UpdatedBy      uint64       `json:"updated_by" gorm:"type:bigint;size:20;"`
	// Status         string       `json:"status" gorm:"type:string;size:50;not null;default:active"`
	Organization Organization `gorm:"foreignKey:OrganizationId"`
	Member       UserAuth     `gorm:"foreignKey:UserAuthId"`
}

type UserProjectRole struct {
	gorm_model.Audited
	gorm_model.Mutable

	UserAuthId uint64 `json:"user_auth_id" gorm:"type:bigint;size:20;not null;index:ur_idx_auth_id"`
	ProjectId  uint64 `json:"project_id" gorm:"type:bigint;size:20;not null"`
	Role       string `json:"role" gorm:"type:string;size:200;not null;"`
	// CreatedBy  uint64   `json:"created_by" gorm:"type:bigint;size:20;not null"`
	// UpdatedBy  uint64   `json:"updated_by" gorm:"type:bigint;size:20"`
	// Status     string   `json:"status" gorm:"type:string;size:50;not null;default:active"`
	Project Project  `gorm:"foreignKey:ProjectId"`
	Member  UserAuth `gorm:"foreignKey:UserAuthId"`
}

func (uor *UserOrganizationRole) GetId() uint64 {
	return uor.Id
}

func (uor *UserOrganizationRole) GetOrganizationId() uint64 {
	return uor.OrganizationId
}

func (uor *UserOrganizationRole) GetRoleName() string {
	return uor.Role
}

func (uor *UserProjectRole) GetId() uint64 {
	return uor.Id
}

func (uor *UserProjectRole) GetProjectId() uint64 {
	return uor.ProjectId
}

func (uor *UserProjectRole) GetRoleName() string {
	return uor.Role
}

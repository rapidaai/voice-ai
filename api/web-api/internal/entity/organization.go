package internal_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type Organization struct {
	gorm_model.Audited
	gorm_model.Mutable
	Name        string `json:"name" gorm:"type:string;size:200;not null"`
	Description string `json:"description" gorm:"type:string;size:400"`
	Size        string `json:"size" gorm:"type:string;size:100"`
	Industry    string `json:"industry" gorm:"type:string;size:200;not null"`
	Contact     string `json:"contact" gorm:"type:string;size:200;not null"`
}

type Vault struct {
	gorm_model.Audited
	gorm_model.Mutable
	//
	VaultType   gorm_types.VaultType `json:"vaultType" gorm:"type:string;size:200;not null"`
	VaultTypeId uint64               `json:"vaultTypeId" gorm:"type:bigint;size:40;not null"`

	VaultLevel   gorm_types.VaultLevel `json:"vaultLevel" gorm:"type:string;size:200;not null"`
	VaultLevelId uint64                `json:"vaultLevelId" gorm:"type:bigint;size:40;not null"`

	Name  string                  `json:"name" gorm:"type:string;size:200;not null"`
	Value gorm_types.InterfaceMap `json:"value" gorm:"type:string;size:50;not null;default:active"`
}

type Project struct {
	gorm_model.Audited
	gorm_model.Mutable
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;size:40;not null"`
	Name           string `json:"name" gorm:"type:string;size:200;not null"`
	Description    string `json:"description" gorm:"type:string;size:400;not null"`
}

type ProjectCredential struct {
	gorm_model.Audited
	gorm_model.Mutable
	ProjectId      uint64   `json:"projectId" gorm:"type:bigint;size:40;not null"`
	OrganizationId uint64   `json:"organizationId" gorm:"type:bigint;size:40;not null"`
	Name           string   `json:"name" gorm:"type:string;size:200;not null"`
	Key            string   `json:"key" gorm:"type:string;size:200;not null"`
	CreatedUser    UserAuth `json:"createdUser" gorm:"foreignKey:CreatedBy"`
}

type OAuthExternalConnect struct {
	gorm_model.Audited
	Identifier  string                `json:"identifier" gorm:"type:string;size:200;not null"`
	ToolConnect string                `json:"toolConnect" gorm:"type:string;size:200;not null"`
	ToolId      uint64                `json:"toolId" gorm:"type:bigint;size:20;not null"`
	LinkerId    uint64                `json:"linkerId" gorm:"type:bigint;size:20;not null"`
	Linker      gorm_types.VaultLevel `json:"linker" gorm:"type:string;size:200;not null"`
	RedirectTo  string                `json:"redirectTo" gorm:"type:string;size:200;not null"`
}

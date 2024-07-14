package internal_gorm

import (
	gorm_model "github.com/lexatic/web-backend/pkg/models/gorm"
	gorm_types "github.com/lexatic/web-backend/pkg/models/gorm/types"
)

type Organization struct {
	gorm_model.Audited
	Name        string `json:"name" gorm:"type:string;size:200;not null"`
	Description string `json:"description" gorm:"type:string;size:400"`
	Size        string `json:"size" gorm:"type:string;size:100"`
	Industry    string `json:"industry" gorm:"type:string;size:200;not null"`
	Contact     string `json:"contact" gorm:"type:string;size:200;not null"`
	Status      string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	CreatedBy   uint64 `json:"createdBy" gorm:"type:bigint;size:200;not null"`
	UpdatedBy   uint64 `json:"updatedBy" gorm:"type:bigint;size:200;not null"`
}

type Vault struct {
	gorm_model.Audited
	//
	VaultType   gorm_types.VaultType `json:"vaultType" gorm:"type:string;size:200;not null"`
	VaultTypeId uint64               `json:"vaultTypeId" gorm:"type:bigint;size:40;not null"`

	VaultLevel   gorm_types.VaultLevel `json:"vaultLevel" gorm:"type:string;size:200;not null"`
	VaultLevelId uint64                `json:"vaultLevelId" gorm:"type:bigint;size:40;not null"`

	Name      string                  `json:"name" gorm:"type:string;size:200;not null"`
	Value     gorm_types.InterfaceMap `json:"value" gorm:"type:string;size:50;not null;default:active"`
	Status    string                  `json:"status" gorm:"type:string;size:50;not null;default:active"`
	CreatedBy uint64                  `json:"createdBy" gorm:"type:bigint;size:200;not null"`
	UpdatedBy uint64                  `json:"updatedBy" gorm:"type:bigint;size:200;not null"`
}

type Project struct {
	gorm_model.Audited
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;size:40;not null"`
	Name           string `json:"name" gorm:"type:string;size:200;not null"`
	Description    string `json:"description" gorm:"type:string;size:400;not null"`
	Status         string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	CreatedBy      uint64 `json:"createdBy" gorm:"type:bigint;size:200;not null"`
	UpdatedBy      uint64 `json:"updatedBy" gorm:"type:bigint;size:200;not null"`
}

type ProjectCredential struct {
	gorm_model.Audited
	ProjectId      uint64 `json:"projectId" gorm:"type:bigint;size:40;not null"`
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;size:40;not null"`
	Name           string `json:"name" gorm:"type:string;size:200;not null"`
	Key            string `json:"key" gorm:"type:string;size:200;not null"`
	Status         string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	CreatedBy      uint64 `json:"createdBy" gorm:"type:bigint;size:200;not null"`
	UpdatedBy      uint64 `json:"updatedBy" gorm:"type:bigint;size:200;not null"`
}

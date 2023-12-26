package internal_gorm

import gorm_model "github.com/lexatic/web-backend/pkg/models/gorm"

type Organization struct {
	gorm_model.Audited
	Name        string `json:"name" gorm:"type:string;size:200;not null"`
	Description string `json:"description" gorm:"type:string;size:400"`
	Size        string `json:"size" gorm:"type:string;size:100"`
	Industry    string `json:"industry" gorm:"type:string;size:200;not null"`
	Contact     string `json:"contact" gorm:"type:string;size:200;not null"`
	Status      string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	CreatedBy   uint64 `json:"created_by" gorm:"type:bigint;size:200;not null"`
	UpdatedBy   uint64 `json:"updated_by" gorm:"type:bigint;size:200;not null"`
}

type Vault struct {
	gorm_model.Audited
	ProviderId     uint64 `json:"providerId" gorm:"type:bigint;size:40;not null"`
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;size:40;not null"`
	Name           string `json:"name" gorm:"type:string;size:200;not null"`
	Key            string `json:"key" gorm:"type:string;size:200;not null"`
	Status         string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	CreatedBy      uint64 `json:"createdBy" gorm:"type:bigint;size:200;not null"`
	UpdatedBy      uint64 `json:"updatedBy" gorm:"type:bigint;size:200;not null"`
}

type Project struct {
	gorm_model.Audited
	OrganizationId uint64 `json:"organization_id" gorm:"type:bigint;size:40;not null"`
	Name           string `json:"name" gorm:"type:string;size:200;not null"`
	Description    string `json:"description" gorm:"type:string;size:400;not null"`
	Status         string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	CreatedBy      uint64 `json:"created_by" gorm:"type:bigint;size:200;not null"`
	UpdatedBy      uint64 `json:"updated_by" gorm:"type:bigint;size:200;not null"`
}

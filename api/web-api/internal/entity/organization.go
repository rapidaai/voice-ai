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
	gorm_model.Organizational
	Provider string                  `json:"provider" gorm:"type:string;size:200;not null"`
	Name     string                  `json:"name" gorm:"type:string;size:200;not null"`
	Value    gorm_types.InterfaceMap `json:"value" gorm:"type:string;size:50;not null;default:active"`
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
	gorm_model.Organizational
	Name string `json:"name" gorm:"type:string;size:200;not null"`
	Key  string `json:"key" gorm:"type:string;size:200;not null"`
}

type OrganizationCredential struct {
	gorm_model.Audited
	gorm_model.Mutable
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;not null"`
	Name           string `json:"name" gorm:"type:string;size:200;not null"`
	Key            string `json:"key" gorm:"type:string;size:200;not null"`
}

type OAuthExternalConnect struct {
	gorm_model.Organizational
	Identifier string `json:"identifier" gorm:"type:string;size:200;not null"`
	Provider   string `json:"provider" gorm:"type:string;size:200;not null"`
	RedirectTo string `json:"redirectTo" gorm:"type:string;size:200;not null"`
}

package internal_entity

import gorm_models "github.com/rapidaai/pkg/models/gorm"

type Lead struct {
	gorm_models.Audited
	Email          string                  `json:"email" gorm:"type:string;size:200;not null;"`
	CompanyName    string                  `json:"companyName" gorm:"type:string;size:200;not null;"`
	ExpectedVolume string                  `json:"expectedVolume" gorm:"type:string;size:200;not null;"`
	CreatedDate    gorm_models.TimeWrapper `json:"createdDate" gorm:"type:timestamp;not null;default:NOW();<-:create"`
}

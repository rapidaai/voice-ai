package internal_gorm

import gorm_model "github.com/lexatic/web-backend/pkg/models/gorm"

type Lead struct {
	gorm_model.Audited
	Email  string `json:"email" gorm:"type:string;size:200;not null"`
	Status string `json:"status" gorm:"type:string;size:50;not null;default:active"`
}

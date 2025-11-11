package internal_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

type EndpointLogArgument struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Argument
	EndpointLogId uint64 `json:"endpointLogId" gorm:"type:bigint;not null"`
}

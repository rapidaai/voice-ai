package internal_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

//

type EndpointLog struct {
	gorm_model.Audited
	gorm_model.Organizational
	EndpointId              uint64                 `json:"endpointId" gorm:"type:bigint;not null"`
	EndpointProviderModelId uint64                 `json:"endpointProviderModelId" gorm:"type:bigint;not null"`
	Source                  string                 `json:"source" gorm:"type:string;size:50;not null;"`
	Status                  type_enums.RecordState `json:"status" gorm:"type:string;size:50;not null;default:ACTIVE"`
	TimeTaken               uint64                 `json:"timeTaken" gorm:"type:bigint;size:20"`

	Arguments []*EndpointLogArgument `json:"arguments" gorm:"foreignKey:EndpointLogId"`
	Metadata  []*EndpointLogMetadata `json:"metadatas" gorm:"foreignKey:EndpointLogId"`
	Options   []*EndpointLogMetadata `json:"options" gorm:"foreignKey:EndpointLogId"`
	Metrics   []*EndpointLogMetric   `json:"metrics" gorm:"foreignKey:EndpointLogId"`
}

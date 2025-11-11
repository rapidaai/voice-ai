package internal_entity

import (
	"strconv"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
)

type ExternalAudit struct {
	gorm_model.Audited
	IntegrationName        string                  `json:"integrationName" gorm:"type:string;size:200;not null"`
	AssetPrefix            string                  `json:"assetPrefix" gorm:"type:string;size:200;not null"`
	ResponseStatus         int64                   `json:"responseStatus" gorm:"type:bigint;size:10"`
	TimeTaken              int64                   `json:"timeTaken" gorm:"type:bigint;size:20"`
	Status                 string                  `json:"status" gorm:"type:string;size:50;not null;default:active"`
	ProjectId              uint64                  `json:"projectId" gorm:"type:bigint"`
	OrganizationId         uint64                  `json:"organizationId" gorm:"type:bigint;not null"`
	CredentialId           uint64                  `json:"credentialId" gorm:"type:bigint;not null"`
	ExternalAuditMetadatas []ExternalAuditMetadata `gorm:"foreignKey:ExternalAuditId"`
	Metrics                gorm_types.MapArray     `json:"metrics" gorm:"type:string"`
}

func (epm *ExternalAudit) SetMetrics(metrics types.Metrics) {
	var result []map[string]string
	epm.ResponseStatus = 200
	for _, metric := range metrics {
		if metric != nil {
			metricMap := map[string]string{
				"name":        metric.GetName(),
				"value":       metric.GetValue(),
				"description": metric.GetDescription(),
			}
			epm.ResponseStatus = 200
			result = append(result, metricMap)
			if metric.GetName() == type_enums.TIME_TAKEN.String() {
				if tt, err := strconv.ParseInt(metric.GetValue(), 10, 64); err == nil {
					epm.TimeTaken = tt
				}
			}
			if metric.GetName() == type_enums.STATUS.String() {
				if metric.GetValue() == "FAILED" {
					epm.ResponseStatus = 500
				}
			}
		}
	}

	epm.Metrics = result
}

type ExternalAuditMetadata struct {
	gorm_model.Audited
	ExternalAuditId uint64 `json:"externalAuditId" gorm:"type:bigint;not null"`
	Key             string `json:"key" gorm:"type:string;size:200;not null"`
	Value           string `json:"value" gorm:"type:string;size:1000;not null"`
}

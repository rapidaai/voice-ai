package internal_entity

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
)

type EndpointLogMetric struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metric
	EndpointLogId uint64 `json:"endpointLogId" gorm:"type:bigint;not null"`
}

// CREATE TABLE endpoint_log_metrics (
//     id BIGINT PRIMARY KEY NOT NULL,
//     status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
//     created_by BIGINT NOT NULL,
//     updated_by BIGINT,
//     created_date TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
//     updated_date TIMESTAMP DEFAULT NULL,
//     name VARCHAR(200) NOT NULL,
//     value TEXT NOT NULL,
// 	description   TEXT,
//     endpoint_log_id BIGINT NOT NULL,
//     CONSTRAINT uk_endpoint_log_id_mtrs UNIQUE (name, endpoint_log_id)
// );

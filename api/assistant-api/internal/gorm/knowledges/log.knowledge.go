package internal_knowledge_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type KnowledgeLog struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Organizational
	KnowledgeId     uint64               `json:"knowledgeId" gorm:"type:bigint"`
	RetrievalMethod string               `json:"retrievalmethod" gorm:"type:string"`
	TopK            uint32               `json:"topK" gorm:"type:int"`
	ScoreThreshold  float32              `json:"scoreThreshold" gorm:"type:float"`
	DocumentCount   int                  `json:"documentCount" gorm:"type:int"`
	AssetPrefix     string               `json:"assetPrefix" gorm:"type:string;size:200;not null"`
	TimeTaken       int64                `json:"timeTaken" gorm:"type:bigint;size:20"`
	AdditionalData  gorm_types.StringMap `json:"additionalData" gorm:"type:string"`
}

//  CREATE TABLE knowledge_logs (
//     id BIGINT PRIMARY KEY,
//     created_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
//     updated_date TIMESTAMP WITH TIME ZONE,
//     status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
//     created_by BIGINT,
//     updated_by BIGINT,
//     project_id BIGINT NOT NULL,
//     organization_id BIGINT NOT NULL,
//     knowledge_id BIGINT NOT NULL,
//     retrieval_method VARCHAR(50),
//     top_k INTEGER,
//     score_threshold REAL,
//     document_count INTEGER,
//     asset_prefix VARCHAR(200) NOT NULL,
//     time_taken BIGINT,
//     additional_data TEXT
// );

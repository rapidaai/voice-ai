// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
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

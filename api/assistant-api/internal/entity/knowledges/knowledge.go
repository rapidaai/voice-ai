// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_knowledge_gorm

import (
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/utils"
)

type Knowledge struct {
	gorm_model.Audited
	gorm_model.Mutable
	Name        string `json:"name" gorm:"type:string"`
	Description string `json:"description" gorm:"type:string"`
	Visibility  string `json:"visibility" gorm:"type:string;default:private"`

	ProjectId      uint64 `json:"projectId" gorm:"type:bigint;size:20;not null"`
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;size:20;not null"`

	KnowledgeTag       *KnowledgeTag        `json:"knowledgeTag" gorm:"foreignKey:KnowledgeId"`
	KnowledgeDocuments []*KnowledgeDocument `json:"knowledgeDocuments" gorm:"foreignKey:KnowledgeId"`
	StorageNamespace   string               `json:"storageNamespace" gorm:"type:string"`

	EmbeddingModelProviderName     string                           `json:"embeddingModelProviderName" gorm:"type:string"`
	KnowledgeEmbeddingModelOptions []*KnowledgeEmbeddingModelOption `json:"knowledgeEmbeddingModelOptions" gorm:"foreignKey:KnowledgeId"`
}

type KnowledgeEmbeddingModelOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	KnowledgeId uint64 `json:"KnowledgeId" gorm:"type:bigint;size:20"`
}

func (a *Knowledge) GetOptions() utils.Option {
	opts := map[string]interface{}{}
	for _, v := range a.KnowledgeEmbeddingModelOptions {
		opts[v.Key] = v.Value
	}
	return opts
}

type KnowledgeTag struct {
	gorm_model.Audited
	KnowledgeId uint64                 `json:"knowledgeId" gorm:"type:bigint;not null"`
	Tag         gorm_types.StringArray `json:"tag" gorm:"type:string;size:200;not null"`
	CreatedBy   uint64                 `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	UpdatedBy   uint64                 `json:"updatedBy" gorm:"type:bigint;size:20;"`
}

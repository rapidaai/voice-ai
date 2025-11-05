package internal_assistant_gorm

import (
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/gorm/knowledges"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type AssistantKnowledge struct {
	gorm_model.Audited
	gorm_model.Mutable
	AssistantId     uint64                     `json:"assistantId" gorm:"type:bigint;size:20"`
	KnowledgeId     uint64                     `json:"knowledgeId" gorm:"type:bigint;size:20"`
	RetrievalMethod gorm_types.RetrievalMethod `json:"retrievalMethod" gorm:"type:string"`

	TopK           uint32                             `json:"topK" gorm:"type:bigint;size:20"`
	ScoreThreshold float32                            `json:"scoreThreshold" gorm:"type:float;size:20"`
	Knowledge      *internal_knowledge_gorm.Knowledge `json:"knowledge" gorm:"foreignKey:KnowledgeId"`

	RerankerEnable                    bool                                `json:"rerankerEnable" gorm:"type:boolean;default:false"`
	RerankerModelProviderName         *string                             `json:"rerankerModelProviderName" gorm:"type:string"`
	RerankerModelProviderId           *uint64                             `json:"rerankerModelProviderId" gorm:"type:bigint;size:20;not null"`
	AssistantKnowledgeRerankerOptions []*AssistantKnowledgeRerankerOption `json:"assistantKnowledgeRerankerOptions" gorm:"foreignKey:AssistantKnowledgeId"`
}

// assistant_knowledge_reranker_options.assistant_knowledge_id
type AssistantKnowledgeRerankerOption struct {
	gorm_model.Audited
	gorm_model.Mutable
	gorm_model.Metadata
	AssistantKnowledgeId uint64 `json:"assistantKnowledgeId" gorm:"type:bigint;size:20"`
}

func (a *AssistantKnowledge) GetOptions() map[string]interface{} {
	opts := map[string]interface{}{}
	for _, v := range a.AssistantKnowledgeRerankerOptions {
		opts[v.Key] = v.Value
	}
	return opts
}

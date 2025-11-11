package internal_knowledge_gorm

import gorm_model "github.com/rapidaai/pkg/models/gorm"

type AssistantDeploymentCatalog struct {
	gorm_model.Audited
	gorm_model.Mutable
	KnowledgeId uint64 `json:"knowledgeId" gorm:"type:bigint;size:20"`
	DocumentId  uint64 `json:"documentId" gorm:"type:bigint;size:20"`
	Language    string `json:"language" gorm:"type:string;size:50;default:english"`
}

type AssistantPluginHelpCenterCatalog struct {
	AssistantDeploymentCatalog
	Question string `json:"description" gorm:"type:string"`
	Answer   string `json:"answer" gorm:"type:string"`
}

type AssistantPluginProductCatalog struct {
	AssistantDeploymentCatalog
	Name        string `json:"name" gorm:"type:string"`
	Description string `json:"description" gorm:"type:string"`
	Image       string `json:"image" gorm:"type:string"`
}

type AssistantPluginArticleCatalog struct {
	AssistantDeploymentCatalog
	Title       string `json:"name" gorm:"type:string"`
	Description string `json:"description" gorm:"type:string"`
	Image       string `json:"image" gorm:"type:string"`
}

package internal_knowledge_gorm

import (
	"encoding/json"

	gorm_model "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
)

type KnowledgeDocument struct {
	gorm_model.Audited
	KnowledgeId uint64 `json:"knowledgeId" gorm:"type:bigint;not null"`
	// to make sure things can get scaled
	ProjectId      uint64 `json:"projectId" gorm:"type:bigint;not null"`
	OrganizationId uint64 `json:"organizationId" gorm:"type:bigint;not null"`

	Language string `json:"language" gorm:"type:string;size:50;default:english"`

	Name         string `json:"name" gorm:"type:string"`
	Description  string `json:"description" gorm:"type:string"`
	DocumentPath string `json:"document_path" gorm:"type:string"`
	//
	DocumentSource gorm_types.DocumentMap `json:"documentSource" gorm:"type:string;not null"`
	DocumentSize   uint64                 `json:"documentSize" gorm:"type:bigint;size:20;not null"`

	// document structure
	DocumentStructure string `json:"DocumentStructure" gorm:"column:document_structure"`

	IndexStatus    string `json:"indexStatus" gorm:"type:string;size:50;not null;default:pending"`
	Status         string `json:"status" gorm:"type:string;size:50;not null;default:active"`
	RetrievalCount uint64 `json:"retrievalCount" gorm:"type:bigint;size:20;default:0"`
	TokenCount     uint64 `json:"tokenCount" gorm:"type:bigint;size:20;default:0"`
	WordCount      uint64 `json:"wordCount" gorm:"type:bigint;size:20;default:0"`
	CreatedBy      uint64 `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	UpdatedBy      uint64 `json:"updatedBy" gorm:"type:bigint;size:20;"`

	IndexingLatency      *float64               `gorm:"column:indexing_latency" json:"indexingLatency,omitempty"`
	CompletedAt          gorm_model.TimeWrapper `gorm:"column:completed_at;default:null" json:"completedAt,omitempty"`
	Error                *string                `gorm:"column:error" json:"error,omitempty"`
	ParsingCompletedAt   gorm_model.TimeWrapper `gorm:"column:parsing_completed_at;default:null" json:"parsingCompletedAt,omitempty"`
	ProcessingStartedAt  gorm_model.TimeWrapper `gorm:"column:processing_started_at;default:null" json:"processingStartedAt,omitempty"`
	CleaningCompletedAt  gorm_model.TimeWrapper `gorm:"column:cleaning_completed_at;default:null" json:"cleaningCompletedAt,omitempty"`
	SplittingCompletedAt gorm_model.TimeWrapper `gorm:"column:splitting_completed_at;default:null" json:"splittingCompletedAt,omitempty"`

	// a structure that defines where the index information is stored
	IndexStruct *string `gorm:"column:index_struct" json:"indexStruct,omitempty"`

	//
	KnowledgeDocumentProcessRule *KnowledgeDocumentProcessRule `gorm:"foreignKey:KnowledgeDocumentId"`
}

func (kd *KnowledgeDocument) DisplayStatus() string {
	status := "available"
	if kd.IndexStatus == "pending" {
		status = "queuing"
	} else if kd.IndexStatus != "completed" && kd.IndexStatus != "error" && kd.IndexStatus != "pending" {
		status = "paused"
	} else if kd.IndexStatus == "parsing" || kd.IndexStatus == "cleaning" || kd.IndexStatus == "splitting" || kd.IndexStatus == "indexing" {
		status = "indexing"
	} else if kd.IndexStatus == "error" {
		status = "error"
	} else if kd.IndexStatus == "completed" && kd.Status == "active" {
		status = "available"
	} else if kd.IndexStatus == "completed" && kd.Status != "active" {
		status = "disabled"
	}

	return status
}

func (kd KnowledgeDocument) MarshalJSON() ([]byte, error) {
	type Alias KnowledgeDocument
	return json.Marshal(&struct {
		Alias
		DisplayStatus string `json:"displayStatus"`
	}{
		Alias:         (Alias)(kd),
		DisplayStatus: kd.DisplayStatus(),
	})
}

type KnowledgeDocumentProcessRule struct {
	gorm_model.Audited
	KnowledgeDocumentId uint64               `json:"knowledgeDocumentId" gorm:"type:bigint;not null"`
	Mode                string               `gorm:"type:varchar(255);not null;default:'automatic'"`
	Rules               gorm_types.StringMap `json:"metrics" gorm:"type:text;not null"`
	//
	CreatedBy uint64 `json:"createdBy" gorm:"type:bigint;size:20;not null"`
	UpdatedBy uint64 `json:"updatedBy" gorm:"type:bigint;size:20;"`
}

type KnowledgeDocumentSegment struct {
	gorm_model.Audited
	KnowledgeDocumentId uint64  `json:"knowledgeDocumentId" gorm:"type:bigint;not null"`
	KnowledgeId         uint64  `json:"knowledgeId" gorm:"type:bigint;not null"`
	Position            int     `gorm:"column:position;not null" json:"position"`
	Content             string  `gorm:"column:content;not null" json:"content"`
	Answer              *string `gorm:"column:answer" json:"answer,omitempty"`
	WordCount           int     `gorm:"column:word_count;not null" json:"wordCount"`
	TokenCount          int     `gorm:"column:token_count;not null" json:"tokenCount"`
	HitCount            int     `gorm:"column:hit_count;not null;default:0" json:"hitCount"`

	// Keywords      []string `gorm:"column:keywords" json:"keywords,omitempty"`
	IndexNodeID   string `gorm:"column:index_node_id" json:"indexNodeId,omitempty"`
	IndexNodeHash string `gorm:"column:index_node_hash" json:"indexNodeHash,omitempty"`

	Enabled    bool                   `gorm:"column:enabled;not null;default:true" json:"enabled"`
	DisabledAt gorm_model.TimeWrapper `gorm:"column:disabled_at" json:"disabledAt,omitempty"`
	DisabledBy string                 `gorm:"column:disabled_by" json:"disabledBy,omitempty"`
	Status     string                 `gorm:"column:status;not null;default:'waiting'" json:"status"`

	IndexingAt  gorm_model.TimeWrapper `gorm:"column:indexing_at" json:"indexingAt,omitempty"`
	CompletedAt gorm_model.TimeWrapper `gorm:"column:completed_at" json:"completedAt,omitempty"`
	Error       *string                `gorm:"column:error" json:"error,omitempty"`
	StoppedAt   gorm_model.TimeWrapper `gorm:"column:stopped_at" json:"stoppedAt,omitempty"`
}

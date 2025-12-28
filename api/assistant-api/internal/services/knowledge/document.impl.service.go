// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package internal_knowledge_service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/rapidaai/api/assistant-api/config"
	internal_knowledge_gorm "github.com/rapidaai/api/assistant-api/internal/entity/knowledges"
	internal_services "github.com/rapidaai/api/assistant-api/internal/services"
	"github.com/rapidaai/pkg/ciphers"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	gorm_types "github.com/rapidaai/pkg/models/gorm/types"
	"github.com/rapidaai/pkg/storages"
	storage_files "github.com/rapidaai/pkg/storages/file-storage"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	protos "github.com/rapidaai/protos"
	"gorm.io/gorm/clause"
)

type knowledgeDocumentService struct {
	config     *config.AssistantConfig
	logger     commons.Logger
	postgres   connectors.PostgresConnector
	opensearch connectors.OpenSearchConnector
	storage    storages.Storage
}

var (
	KNOWLEDGE_DOCUMENT_PREFIX = "knowledge-document__"
)

func NewKnowledgeDocumentService(config *config.AssistantConfig, logger commons.Logger, postgres connectors.PostgresConnector, opensearch connectors.OpenSearchConnector) internal_services.KnowledgeDocumentService {
	return &knowledgeDocumentService{
		config:     config,
		logger:     logger,
		postgres:   postgres,
		opensearch: opensearch,
		storage:    storage_files.NewStorage(config.AssetStoreConfig, logger),
	}
}

func (knowledge *knowledgeDocumentService) GetCounts(ctx context.Context, auth types.SimplePrinciple, knowledgeId uint64) (documentCount, wordCount, tokenCount uint32) {
	var result struct {
		DocumentCount   uint32
		TotalTokenCount uint32
		TotalWordCount  uint32
	}

	db := knowledge.postgres.DB(ctx)
	tx := db.Model(&internal_knowledge_gorm.KnowledgeDocument{}).
		Select("COUNT(*) as document_count, SUM(token_count) as total_token_count, SUM(word_count) as total_word_count").
		Where("knowledge_id = ?", knowledgeId).
		Where("project_id = ? ", *auth.GetCurrentProjectId()).
		Where("organization_id = ? ", *auth.GetCurrentOrganizationId()).
		Scan(&result)

	if tx.Error != nil {
		knowledge.logger.Debugf("unable to find any knowledge for given project %v and organization  %v", *auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId())
		return 0, 0, 0
	}
	return result.DocumentCount, result.TotalWordCount, result.TotalTokenCount

}

func (knowledge *knowledgeDocumentService) GetAll(ctx context.Context, auth types.SimplePrinciple,
	knowledgeId uint64,
	criterias []*protos.Criteria, paginate *protos.Paginate) (int64, *[]internal_knowledge_gorm.KnowledgeDocument, error) {
	db := knowledge.postgres.DB(ctx)
	var knowledgeDocuments []internal_knowledge_gorm.KnowledgeDocument
	var cnt int64
	qry := db.Model(internal_knowledge_gorm.KnowledgeDocument{}).
		Where("knowledge_id = ? AND status = ?", knowledgeId, "active")
	for _, ct := range criterias {
		qry.Where(fmt.Sprintf("%s = ?", ct.GetKey()), ct.GetValue())
	}
	tx := qry.
		Scopes(gorm_models.
			Paginate(gorm_models.
				NewPaginated(
					int(paginate.GetPage()),
					int(paginate.GetPageSize()),
					&cnt,
					qry))).
		Order(clause.OrderByColumn{
			Column: clause.Column{Name: "created_date"},
			Desc:   true,
		}).
		Find(&knowledgeDocuments)
	if tx.Error != nil {
		knowledge.logger.Debugf("unable to find any knowledge for given project %v and organization  %v", *auth.GetCurrentProjectId(), *auth.GetCurrentOrganizationId())
		return cnt, nil, tx.Error
	}

	return cnt, &knowledgeDocuments, nil
}

func (knowledge *knowledgeDocumentService) Get(ctx context.Context, auth types.SimplePrinciple, knowledgeId uint64, knowledgeDocumentId uint64) (*internal_knowledge_gorm.KnowledgeDocument, error) {
	db := knowledge.postgres.DB(ctx)
	var _knowledge internal_knowledge_gorm.KnowledgeDocument
	tx := db.
		Where("id = ? AND knowledge_id = ? AND status = ?", knowledgeDocumentId, knowledgeId, "active").
		First(&_knowledge)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &_knowledge, nil
}

func (knowledgeDocument *knowledgeDocumentService) CreateToolDocument(ctx context.Context,
	auth types.SimplePrinciple,
	knowledge *internal_knowledge_gorm.Knowledge,
	datasource string,
	documentStructure string,
	contents []*protos.Content,
) ([]*internal_knowledge_gorm.KnowledgeDocument, error) {
	db := knowledgeDocument.postgres.DB(ctx)
	allKnowledge := make([]*internal_knowledge_gorm.KnowledgeDocument, 0)
	for _, cntnt := range contents {
		allKnowledge = append(allKnowledge, &internal_knowledge_gorm.KnowledgeDocument{
			KnowledgeId:       knowledge.Id,
			Name:              cntnt.GetName(),
			ProjectId:         *auth.GetCurrentProjectId(),
			OrganizationId:    *auth.GetCurrentOrganizationId(),
			CreatedBy:         *auth.GetUserId(),
			DocumentStructure: documentStructure,
			DocumentSize:      0,
			DocumentPath:      cntnt.GetName(),
			DocumentSource: map[string]interface{}{
				"completePath": cntnt.GetName(),
				"documentUrl":  cntnt.GetName(),
				"source":       datasource,
				"type":         "tool",
				"mimeType":     cntnt.GetContentType(),
				"extras":       cntnt.GetMeta().AsMap(),
			},
		})
	}

	tx := db.Create(allKnowledge)
	if tx.Error != nil {
		knowledgeDocument.logger.Errorf("unable to create assistant with error %+v", tx.Error)
		return nil, tx.Error
	}
	return allKnowledge, nil
}

func (knowledgeDocument *knowledgeDocumentService) CreateManualDocument(
	ctx context.Context,
	auth types.SimplePrinciple,
	knowledge *internal_knowledge_gorm.Knowledge,
	datasource string,
	documentStructure string,
	contents []*protos.Content,
) ([]*internal_knowledge_gorm.KnowledgeDocument, error) {

	db := knowledgeDocument.postgres.DB(ctx)
	allKnowledge := make([]*internal_knowledge_gorm.KnowledgeDocument, 0)

	switch datasource {
	case "manual-file":
		for _, cntnt := range contents {
			fileName := fmt.Sprintf("%d/%d/%d_%s%s",
				*auth.GetCurrentOrganizationId(),
				*auth.GetCurrentProjectId(),
				*auth.GetUserId(),
				ciphers.RandomHash(KNOWLEDGE_DOCUMENT_PREFIX), path.Ext(cntnt.GetName()))

			fileContent := cntnt.GetContent()
			knowledgeDocument.logger.Debugf("Uploading document with fileName %s", fileName)
			storageResponse := knowledgeDocument.storage.Store(context.Background(), fileName, fileContent)
			if storageResponse.Error != nil {
				knowledgeDocument.logger.Debugf("unable to upload the content to s3 with error %v", storageResponse.Error)
				continue
			}

			// in case typescript conn't identify the content type i am goging to identify
			contentType := cntnt.GetContentType()
			if contentType == "" {
				contentType = http.DetectContentType(fileContent[:512])
			}

			allKnowledge = append(allKnowledge, &internal_knowledge_gorm.KnowledgeDocument{
				KnowledgeId:       knowledge.Id,
				Name:              cntnt.GetName(),
				ProjectId:         *auth.GetCurrentProjectId(),
				OrganizationId:    *auth.GetCurrentOrganizationId(),
				CreatedBy:         *auth.GetUserId(),
				DocumentSize:      0,
				DocumentStructure: documentStructure,
				DocumentPath:      storageResponse.CompletePath,
				DocumentSource: map[string]interface{}{
					"completePath": storageResponse.CompletePath,
					"documentUrl":  fileName,
					"source":       gorm_types.DOCUMENT_SOURCE_MANUAL,
					"type":         gorm_types.DOCUMENT_SOURCE_MANUAL_FILE,
					"mimeType":     contentType,
					"storage":      storageResponse.StorageType,
				},
			})
		}
	case "manual-url":
		for _, cntnt := range contents {
			origUrl := cntnt.GetName()
			parsedURL, err := url.Parse(origUrl)
			if err != nil {
				knowledgeDocument.logger.Errorf("not able to parse the url as manual url %v", err)
				continue
			}
			if parsedURL.Scheme == "" {
				origUrl = fmt.Sprintf("https://%s", origUrl)
			}

			allKnowledge = append(allKnowledge, &internal_knowledge_gorm.KnowledgeDocument{
				KnowledgeId:       knowledge.Id,
				Name:              cntnt.GetName(),
				ProjectId:         *auth.GetCurrentProjectId(),
				OrganizationId:    *auth.GetCurrentOrganizationId(),
				CreatedBy:         *auth.GetUserId(),
				DocumentPath:      cntnt.GetName(),
				DocumentStructure: documentStructure,
				DocumentSize:      0,
				DocumentSource: map[string]interface{}{
					"documentUrl": origUrl,
					"source":      gorm_types.DOCUMENT_SOURCE_MANUAL,
					"type":        gorm_types.DOCUMENT_SOURCE_MANUAL_URL,
					"mimeType":    knowledgeDocument.webfileMimeType(origUrl, cntnt.GetContentType()),
				},
			})
		}
		// return nil, fmt.Errorf("unsupported datasource currently we support only manual upload of url and files")
	case "manual-zip":
		return nil, fmt.Errorf("unsupported datasource currently we support only manual upload of url and files")
	}

	if len(allKnowledge) == 0 {
		knowledgeDocument.logger.Errorf("unable to create knowledge document as slice of document is empty")
		return nil, fmt.Errorf("unable to create knowledge document")
	}
	tx := db.Create(allKnowledge)
	if tx.Error != nil {
		knowledgeDocument.logger.Errorf("unable to creating knowledge document with error %+v", tx.Error)
		return nil, tx.Error
	}
	return allKnowledge, nil

}

func (knowledge *knowledgeDocumentService) webfileMimeType(
	weburl string, fallback string) string {
	resp, err := http.Head(weburl)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		return fallback
	} else {
		return strings.Split(mimeType, ";")[0]
	}
}

// func (kr *knowledgeDocumentService) getCollectionName(name string) string {
// 	if kr.config.IsDevelopment() {
// 		return fmt.Sprintf("%s__%s", "dev", name)
// 	}
// 	return fmt.Sprintf("%s__%s", "prod", name)
// }

func (knowledge *knowledgeDocumentService) GetAllDocumentSegment(
	ctx context.Context,
	auth types.SimplePrinciple,
	knowledgeId uint64,
	storageNamespace string,
	criterias []*protos.Criteria,
	paginate *protos.Paginate) (int64, []*protos.KnowledgeDocumentSegment, error) {
	indexs := make([]string, 0)
	indexs = append(indexs, storageNamespace)
	// Construct the OpenSearch query
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]uint64{
							"metadata.knowledge_id": knowledgeId,
						},
					},
					{
						"term": map[string]uint64{
							"metadata.project_id": *auth.GetCurrentProjectId(),
						},
					},
					{
						"term": map[string]uint64{
							"metadata.organization_id": *auth.GetCurrentOrganizationId(),
						},
					},
				},
			},
		},
		"_source": map[string]interface{}{
			"includes": []string{
				"document_hash",
				"document_id",
				"text",
				"metadata",
				"entities",
			},
		},
	}

	// Add additional criteria to the query
	for _, ct := range criterias {
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = append(
			query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]map[string]interface{}),
			map[string]interface{}{
				"term": map[string]interface{}{
					fmt.Sprintf("metadata.%s", ct.GetKey()): ct.GetValue(),
				},
			},
		)
	}

	// Set up pagination
	from := (paginate.GetPage() - 1) * paginate.GetPageSize()
	size := paginate.GetPageSize()

	// Add pagination to the query
	query["from"] = from
	query["size"] = size

	searchBodyJSON, err := json.Marshal(query)
	if err != nil {
		knowledge.logger.Errorf("Error marshaling search body: %v", err)
		return 0, nil, err
	}

	result := knowledge.opensearch.SearchWithCount(ctx, indexs, string(searchBodyJSON))
	if result.Err != nil {
		knowledge.logger.Errorf("Error while searching opensearch: %v", err)
		return 0, nil, err
	}

	segments := make([]*protos.KnowledgeDocumentSegment, 0)
	for _, hit := range result.Hits.Hits {
		index, _ := hit["_index"].(string)
		source, ok := hit["_source"].(map[string]interface{})
		if !ok {
			continue
		}

		segment := &protos.KnowledgeDocumentSegment{}
		config := &mapstructure.DecoderConfig{
			TagName: "json",
			Result:  segment,
		}
		decoder, _ := mapstructure.NewDecoder(config)
		if err := decoder.Decode(source); err != nil {
			knowledge.logger.Errorf("failed to decode segment: %v", err)
			continue
		}

		segment.Index = index
		segments = append(segments, segment)
	}
	return int64(result.Hits.Total), segments, nil
}

func (knowledge *knowledgeDocumentService) UpdateDocumentSegment(
	ctx context.Context,
	auth types.SimplePrinciple,
	index string,
	documentId string,
	documentName string,
	organizations []string,
	dates []string,
	products []string,
	events []string,
	people []string,
	times []string,
	quantities []string,
	locations []string,
	industries []string,
) (*protos.KnowledgeDocumentSegment, error) {
	// Construct the update query
	updateQuery := map[string]interface{}{
		"doc": map[string]interface{}{},
	}

	// Add metadata if documentName is not empty
	if documentName != "" {
		updateQuery["doc"].(map[string]interface{})["metadata"] = map[string]interface{}{
			"document_name": documentName,
		}
	}

	// Create entities map
	entities := make(map[string]interface{})

	// Add non-empty entity fields
	if len(organizations) > 0 {
		entities["organizations"] = organizations
	}
	if len(dates) > 0 {
		entities["dates"] = dates
	}
	if len(products) > 0 {
		entities["products"] = products
	}
	if len(events) > 0 {
		entities["events"] = events
	}
	if len(people) > 0 {
		entities["people"] = people
	}
	if len(times) > 0 {
		entities["times"] = times
	}
	if len(quantities) > 0 {
		entities["quantities"] = quantities
	}
	if len(locations) > 0 {
		entities["locations"] = locations
	}
	if len(industries) > 0 {
		entities["industries"] = industries
	}

	// Add entities to updateQuery if not empty
	if len(entities) > 0 {
		updateQuery["doc"].(map[string]interface{})["entities"] = entities
	}

	updateBodyJSON, err := json.Marshal(updateQuery)
	if err != nil {
		knowledge.logger.Errorf("Error marshaling update body: %s", err)
		return nil, err
	}

	err = knowledge.opensearch.Update(ctx, index, documentId, string(updateBodyJSON))
	if err != nil {
		knowledge.logger.Errorf("Error updating document segment: %s", err)
		return nil, err
	}
	return nil, nil
}

func (knowledge *knowledgeDocumentService) DeleteDocumentSegment(
	ctx context.Context,
	auth types.SimplePrinciple,
	index string,
	documentId string,
	reason string,
) (*protos.KnowledgeDocumentSegment, error) {
	// Update the document status directly
	updateBody := map[string]interface{}{
		"doc": map[string]interface{}{
			"status":          type_enums.RECORD_ARCHIEVE.String(), // Assuming there's a status field, adjust as needed
			"archieve_reason": reason,
		},
	}
	updateBodyJSON, err := json.Marshal(updateBody)
	if err != nil {
		knowledge.logger.Errorf("Error marshaling update body: %s", err)
		return nil, err
	}

	err = knowledge.opensearch.Update(ctx, index, documentId, string(updateBodyJSON))
	if err != nil {
		knowledge.logger.Errorf("Error updating document segment: %s", err)
		return nil, err
	}
	return nil, nil
}

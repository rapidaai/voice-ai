package connectors

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	opensearch "github.com/opensearch-project/opensearch-go/v2"
	opensearchapi "github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	commons "github.com/rapidaai/pkg/commons"
	configs "github.com/rapidaai/pkg/configs"
)

type OpenSearchResponse struct {
	Err      error
	Took     int
	Timedout bool
}

type SearchResponseWithCount struct {
	OpenSearchResponse
	Hits struct {
		Total    int
		MaxScore string
		Hits     []map[string]interface{}
	}
}

type SearchResponse struct {
	OpenSearchResponse
	Hits struct {
		Total struct {
			Value    int
			Relation string
		}
		MaxScore string
		Hits     []map[string]interface{}
	}
}

func (osr *OpenSearchResponse) Error() error {
	return osr.Err
}

type OpenSearchConnector interface {
	VectorConnector
	Search(context.Context, []string, string) *SearchResponse
	SearchWithCount(context.Context, []string, string) *SearchResponseWithCount
	Persist(ctx context.Context, index string, id string, body string) error
	Update(ctx context.Context, index string, id string, body string) error
	Bulk(ctx context.Context, body string) error
}

type openSearchConnector struct {
	cfg        *configs.OpenSearchConfig
	Connection *opensearch.Client
	logger     commons.Logger
}

// HybridSearch implements OpenSearchConnector.
func (osc *openSearchConnector) HybridSearch(ctx context.Context,
	collectionName string,
	query string,
	queryVector []float64,
	entities map[string]interface{},
	opts *VectorSearchOptions) ([]map[string]interface{}, error) {
	osc.logger.Debugf("query %s, entities %+v", query, entities)

	// Base query with knn + text match
	opensearchQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []interface{}{
							map[string]interface{}{
								"knn": map[string]interface{}{
									"vector": map[string]interface{}{
										"vector": queryVector,
										"k":      opts.TopK,
									},
								},
							},
							map[string]interface{}{
								"match_phrase": map[string]interface{}{
									"text": map[string]interface{}{
										"query": query,
									},
								},
							},
						},
						"minimum_should_match": 1,
					},
				},
			},
		},
	}

	boolQuery := opensearchQuery["bool"].(map[string]interface{})
	filter := []interface{}{}
	boostShould := []interface{}{}

	for key, val := range entities {
		switch key {
		case "project_id", "organization_id", "document_id", "knowledge_id":
			if id, ok := val.(string); ok && id != "" {
				filter = append(filter, map[string]interface{}{
					"term": map[string]interface{}{
						fmt.Sprintf("metadata.%s", key): id,
					},
				})
			}
		case "document_name", "category":
			if id, ok := val.(string); ok && id != "" {
				filter = append(filter, map[string]interface{}{
					"match": map[string]interface{}{
						fmt.Sprintf("metadata.%s", key): id,
					},
				})
			}
		case "text":
			if textQuery, ok := val.(string); ok && textQuery != "" {
				filter = append(filter, map[string]interface{}{
					"match": map[string]interface{}{
						"text": textQuery,
					},
				})
			}
		case "organization", "dates", "products", "events", "people", "times", "quantities":
			var esField string
			switch key {
			case "organizations":
				esField = "entities.organizations.keyword"
			case "dates":
				esField = "entities.dates.keyword"
			case "products":
				esField = "entities.products.keyword"
			case "events":
				esField = "entities.events.keyword"
			case "people":
				esField = "entities.people.keyword"
			case "times":
				esField = "entities.times.keyword"
			case "quantities":
				esField = "entities.quantities.keyword"
			}

			var terms []interface{}
			switch v := val.(type) {
			case []interface{}:
				for _, vi := range v {
					if s, ok := vi.(string); ok && s != "" {
						terms = append(terms, s)
					}
				}
			case string:
				if v != "" {
					terms = append(terms, v)
				}
			case []string:
				for _, s := range v {
					if s != "" {
						terms = append(terms, s)
					}
				}
			}

			if len(terms) > 0 {
				boostShould = append(boostShould, map[string]interface{}{
					"terms": map[string]interface{}{
						esField: terms,
					},
				})
			}
		}
	}

	// Add filters if present
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}

	// Add boost clauses to should array inside the first must bool
	if len(boostShould) > 0 {
		mustBool := boolQuery["must"].([]interface{})[0].(map[string]interface{})["bool"].(map[string]interface{})
		existingShould, ok := mustBool["should"].([]interface{})
		if !ok {
			existingShould = []interface{}{}
		}
		mustBool["should"] = append(existingShould, boostShould...)
	}

	// Build search body
	searchBody := map[string]interface{}{
		"min_score": opts.MinScore,
		"_source":   opts.Source,
		"query":     opensearchQuery,
		"size":      opts.TopK,
	}

	searchBodyJSON, err := json.Marshal(searchBody)
	if err != nil {
		osc.logger.Fatalf("Error marshaling search body: %s", err)
		return nil, err
	}

	result := osc.Search(ctx, []string{collectionName}, string(searchBodyJSON))
	return result.Hits.Hits, result.Error()
}

func (osc *openSearchConnector) TextSearch(ctx context.Context,
	collectionName string, query string,
	entities map[string]interface{},
	opts *VectorSearchOptions) ([]map[string]interface{}, error) {

	opensearchQuery := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"match_phrase": map[string]interface{}{
						"text": query,
					},
				},
			},
		},
	}

	boolQuery := opensearchQuery["bool"].(map[string]interface{})
	filter := []interface{}{}
	boostShould := []interface{}{}

	for key, val := range entities {
		switch key {
		case "project_id", "organization_id", "document_id", "knowledge_id", "document_name", "category":
			if id, ok := val.(string); ok && id != "" {
				filter = append(filter, map[string]interface{}{
					"term": map[string]interface{}{
						fmt.Sprintf("metadata.%s", key): id,
					},
				})
			}
		case "text":
			if textQuery, ok := val.(string); ok && textQuery != "" {
				filter = append(filter, map[string]interface{}{
					"match": map[string]interface{}{
						"text": textQuery,
					},
				})
			}
		case "organization", "dates", "products", "events", "people", "times", "quantities":
			var esField string
			switch key {
			case "organizations":
				esField = "entities.organizations.keyword"
			case "dates":
				esField = "entities.dates.keyword"
			case "products":
				esField = "entities.products.keyword"
			case "events":
				esField = "entities.events.keyword"
			case "people":
				esField = "entities.people.keyword"
			case "times":
				esField = "entities.times.keyword"
			case "quantities":
				esField = "entities.quantities.keyword"
			}

			var terms []interface{}
			switch v := val.(type) {
			case []interface{}:
				for _, vi := range v {
					if s, ok := vi.(string); ok && s != "" {
						terms = append(terms, s)
					}
				}
			case string:
				if v != "" {
					terms = append(terms, v)
				}
			case []string:
				for _, s := range v {
					if s != "" {
						terms = append(terms, s)
					}
				}
			}

			if len(terms) > 0 {
				boostShould = append(boostShould, map[string]interface{}{
					"terms": map[string]interface{}{
						esField: terms,
					},
				})
			}
		}
	}

	// Add filters if present
	if len(filter) > 0 {
		boolQuery["filter"] = filter
	}

	// Add boostShould clauses at the root `should` level
	if len(boostShould) > 0 {
		boolQuery["should"] = boostShould
	}

	searchBody := map[string]interface{}{
		"min_score": opts.MinScore,
		"_source":   opts.Source,
		"query":     opensearchQuery,
		"size":      opts.TopK,
	}

	searchBodyJSON, err := json.Marshal(searchBody)
	if err != nil {
		osc.logger.Fatalf("Error marshaling search body: %s", err)
		return nil, err
	}

	result := osc.Search(ctx, []string{collectionName}, string(searchBodyJSON))
	return result.Hits.Hits, result.Error()
}

// VectorSearch implements OpenSearchConnector.
func (osc *openSearchConnector) VectorSearch(ctx context.Context,
	collectionName string,
	queryVector []float64,
	entities map[string]interface{},
	opts *VectorSearchOptions) ([]map[string]interface{}, error) {

	// Initial KNN query
	knnQuery := map[string]interface{}{
		"vector": map[string]interface{}{
			"vector": queryVector,
			"k":      opts.TopK,
		},
	}

	// Bool query parts
	filter := []interface{}{}
	should := []interface{}{}

	// Build filters and should clauses
	for key, val := range entities {
		switch key {
		case "project_id", "organization_id", "document_id", "knowledge_id", "document_name", "category":
			if id, ok := val.(string); ok && id != "" {
				filter = append(filter, map[string]interface{}{
					"term": map[string]interface{}{
						fmt.Sprintf("metadata.%s", key): id,
					},
				})
			}
		case "text":
			if textQuery, ok := val.(string); ok && textQuery != "" {
				filter = append(filter, map[string]interface{}{
					"match": map[string]interface{}{
						"text": textQuery,
					},
				})
			}
		case "organization", "dates", "products", "events", "people", "times", "quantities":
			var esField string
			switch key {
			case "organizations":
				esField = "entities.organizations.keyword"
			case "dates":
				esField = "entities.dates.keyword"
			case "products":
				esField = "entities.products.keyword"
			case "events":
				esField = "entities.events.keyword"
			case "people":
				esField = "entities.people.keyword"
			case "times":
				esField = "entities.times.keyword"
			case "quantities":
				esField = "entities.quantities.keyword"
			}

			var terms []interface{}
			switch v := val.(type) {
			case []interface{}:
				for _, vi := range v {
					if s, ok := vi.(string); ok && s != "" {
						terms = append(terms, s)
					}
				}
			case string:
				if v != "" {
					terms = append(terms, v)
				}
			case []string:
				for _, s := range v {
					if s != "" {
						terms = append(terms, s)
					}
				}
			}

			if len(terms) > 0 {
				should = append(should, map[string]interface{}{
					"terms": map[string]interface{}{
						esField: terms,
					},
				})
			}
		}
	}

	var finalQuery map[string]interface{}

	// If no filters or should clauses, use KNN directly
	if len(filter) == 0 && len(should) == 0 {
		finalQuery = map[string]interface{}{
			"knn": knnQuery,
		}
	} else {
		// Otherwise, wrap knn inside a bool.must clause
		boolQuery := map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"knn": knnQuery,
				},
			},
		}
		if len(filter) > 0 {
			boolQuery["filter"] = filter
		}
		if len(should) > 0 {
			boolQuery["should"] = should
		}
		finalQuery = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	// Construct final search body
	searchBody := map[string]interface{}{
		"query":     finalQuery,
		"size":      opts.TopK,
		"min_score": opts.MinScore,
		"_source":   opts.Source,
	}

	searchBodyJSON, err := json.Marshal(searchBody)
	if err != nil {
		osc.logger.Errorf("Error marshaling search body: %s", err)
		return nil, err
	}

	result := osc.Search(ctx, []string{collectionName}, string(searchBodyJSON))
	if result.Error() != nil {
		osc.logger.Errorf("Search error: %s", result.Error())
	}
	return result.Hits.Hits, result.Error()
}

// return connector behavior for opensearch
func NewOpenSearchConnector(config *configs.OpenSearchConfig, logger commons.Logger) OpenSearchConnector {
	return &openSearchConnector{cfg: config, logger: logger}
}

// generating connection string from configuration
func (openSearch *openSearchConnector) connectionString() string {
	return fmt.Sprintf("%s://%s", openSearch.cfg.Schema, openSearch.cfg.Host)
}

// connecting and setting connection for opensearch
func (openSearch *openSearchConnector) Connect(ctx context.Context) error {

	openSearch.logger.Debugf("Creating opensearch client %s", openSearch.connectionString())
	client, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			MaxConnsPerHost: openSearch.cfg.MaxConnection,
		},
		Username:      openSearch.cfg.Auth.User,
		Password:      openSearch.cfg.Auth.Password,
		Addresses:     []string{openSearch.connectionString()},
		MaxRetries:    openSearch.cfg.MaxRetries,
		RetryOnStatus: []int{502, 503, 504},
	})
	if err != nil {
		return err
	}
	openSearch.logger.Debugf("Created the client for opensearch %s", openSearch.connectionString())
	openSearch.Connection = client
	return nil
}

// name for connector maybe logging or debug purposes
func (openSearch *openSearchConnector) Name() string {
	return fmt.Sprintf("ES %s://%s", openSearch.cfg.Schema, openSearch.cfg.Host)
}

// call info and check if the connection can be establish for any opensearch operation
func (openSearch *openSearchConnector) IsConnected(ctx context.Context) bool {

	openSearch.logger.Debugf("Calling info for opensearch.")
	infoQuery := opensearchapi.InfoRequest{
		ErrorTrace: true,
	}
	infoResponse, err := infoQuery.Do(ctx, openSearch.Connection)
	if err != nil {
		return false
	}
	defer infoResponse.Body.Close()

	openSearch.logger.Debugf("Completed info call for opensearch.")
	// some case opensearch do not raise any error when using with aws sts and role base IAM authentication
	if infoResponse.StatusCode != 200 {
		openSearch.logger.Debugf("Recieve the response from opensearch for INFO %v", infoResponse)
		return false
	}
	openSearch.logger.Debugf("Returning info for opensearch with connected state.")
	return true
}

// return searchresponse from search query on given index and body with overall count
func (openSearch *openSearchConnector) SearchWithCount(ctx context.Context, index []string, body string) *SearchResponseWithCount {
	searchResponse := &SearchResponseWithCount{}
	openSearch.logger.Debugf("searching with count query on index %s and query body %s", index, body)
	err := openSearch.search(ctx, index, body, true, searchResponse)
	if err != nil {
		searchResponse.Err = err
		return searchResponse
	}
	openSearch.logger.Infof("returning opensearch `SearchWithCount` result with count %d time %v", searchResponse.Hits.Total, searchResponse.Took)
	return searchResponse
}

// return searchresponse from search query on given index and body
func (openSearch *openSearchConnector) Search(ctx context.Context, index []string, body string) *SearchResponse {
	searchResponse := &SearchResponse{}
	openSearch.logger.Debugf("searching query on index %s with query %s", index, body)
	err := openSearch.search(ctx, index, body, false, searchResponse)
	if err != nil {
		openSearch.logger.Errorf("error while searching the document at opensearch %v", err)
		searchResponse.Err = err
		return searchResponse
	}

	openSearch.logger.Infof("returning opensearch `Search` result with count %d time %v", searchResponse.Hits.Total.Value, searchResponse.Took)
	return searchResponse
}

// raw search execution for open search
func (openSearch *openSearchConnector) search(ctx context.Context, index []string, body string, totalHitsAsInt bool, output interface{}) error {
	// only for benchmarking
	start := time.Now()

	openSearch.logger.Debugf("searching query started executing on index %s", index)
	searchQuery := opensearchapi.SearchRequest{
		Index:              index,
		Body:               strings.NewReader(body),
		RestTotalHitsAsInt: &totalHitsAsInt,
	}
	searchResponse, err := searchQuery.Do(ctx, openSearch.Connection)
	if err != nil {
		return err
	}
	openSearch.logger.Infof("querying opensearch `internal/search` time %v", time.Since(start))
	defer searchResponse.Body.Close()
	if searchResponse.IsError() {
		openSearch.logger.Errorf("error searching to opensearch status is not legal: %v complete response %+v", searchResponse.StatusCode, searchResponse)
		return err
	}
	err = json.NewDecoder(searchResponse.Body).Decode(&output)
	if err != nil {
		openSearch.logger.Errorf("unable to unmarshal response from open search. %v", err)
		return err
	}
	openSearch.logger.Debugf("searching query completed executing on index %s and result %v", index, searchResponse)
	openSearch.logger.Infof("returning opensearch `internal/search` result time %v", time.Since(start))
	return nil
}

// bulk operation body should contain complete information about action
func (openSearch *openSearchConnector) Bulk(ctx context.Context, body string) error {
	openSearch.logger.Debugf("bulk operation started with body %s", body)
	req := opensearchapi.BulkRequest{
		Body:    strings.NewReader(body),
		Refresh: "true",
	}
	bulkResponse, err := req.Do(context.Background(), openSearch.Connection)
	if err != nil {
		openSearch.logger.Errorf("error while bulk operation to opensearch got error %v", err)
		return err
	}
	defer bulkResponse.Body.Close()
	openSearch.logger.Debugf("response from open search %v", bulkResponse)
	if bulkResponse.IsError() {
		openSearch.logger.Errorf("error while bulk operation to opensearch status is not legal: %v", bulkResponse.StatusCode)
		return err
	}
	openSearch.logger.Debugf("bulk operation completed executing with body %s", body)
	return nil
}

// persisting body to index in opensearch
func (openSearch *openSearchConnector) Persist(ctx context.Context, index string, id string, body string) error {

	openSearch.logger.Debugf("indexing query started executing on index %s", index)
	req := opensearchapi.IndexRequest{
		Index:      index,
		Body:       strings.NewReader(body),
		DocumentID: id,
	}
	insertResponse, err := req.Do(ctx, openSearch.Connection)
	if err != nil {
		openSearch.logger.Errorf("error persisting to opensearch index %s got error %v", index, err)
		return err
	}
	defer insertResponse.Body.Close()
	openSearch.logger.Debugf("response from open search %v", insertResponse)
	if insertResponse.IsError() {
		openSearch.logger.Errorf("error persisting to opensearch status is not legal: %v", insertResponse.StatusCode)
		return err
	}
	openSearch.logger.Debugf("indexing query completed executing on index %s", index)
	return nil
}

func (openSearch *openSearchConnector) Update(ctx context.Context, index string, id string, body string) error {
	openSearch.logger.Debugf("update query started executing on index %s", index)
	req := opensearchapi.UpdateRequest{
		Index:      index,
		Body:       strings.NewReader(body),
		DocumentID: id,
		Refresh:    "true",
	}
	updateResponse, err := req.Do(ctx, openSearch.Connection)

	if err != nil {
		openSearch.logger.Errorf("error persisting to opensearch index %s got error %v", index, err)
		return err
	}
	defer updateResponse.Body.Close()
	openSearch.logger.Debugf("response from open search %v", updateResponse)
	if updateResponse.IsError() {
		openSearch.logger.Errorf("error persisting to opensearch status is not legal: %v", updateResponse.StatusCode)
		return err
	}
	openSearch.logger.Debugf("update query completed executing on index %s", index)
	return nil
}

// disconnect from opensearch client
func (c *openSearchConnector) Disconnect(ctx context.Context) error {

	// do somthing to close the connection
	// defer c.Connection.close()
	c.logger.Debug("Disconnecting with opensearch client.")
	c.Connection = nil
	return nil
}

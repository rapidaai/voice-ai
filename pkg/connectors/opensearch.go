package connectors

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	commons "github.com/lexatic/web-backend/pkg/commons"
	configs "github.com/lexatic/web-backend/pkg/configs"
	mapstructure "github.com/mitchellh/mapstructure"
	opensearch "github.com/opensearch-project/opensearch-go"
	opensearchapi "github.com/opensearch-project/opensearch-go/opensearchapi"
)

type OpenSearchConnector interface {
	Connector
	Search(context.Context, string, string) *SearchResponse
	SearchWithCount(context.Context, string, string) *SearchResponseWithCount
	Persist(ctx context.Context, index string, id string, body string) error
	Bulk(ctx context.Context, body string) error
}

type openSearchConnector struct {
	cfg        *configs.OpenSearchConfig
	Connection *opensearch.Client
	logger     commons.Logger
}

// return connector behavior for opensearch
func NewOpenSearchConnector(config *configs.OpenSearchConfig, logger commons.Logger) OpenSearchConnector {
	return &openSearchConnector{cfg: config, logger: logger}
}

// generating connection string from configuration
func (openSearch *openSearchConnector) connectionString() string {
	if openSearch.cfg.Port > 0 {
		return fmt.Sprintf("%s://%s:%d", openSearch.cfg.Schema, openSearch.cfg.Host, openSearch.cfg.Port)
	}
	return fmt.Sprintf("%s://%s", openSearch.cfg.Schema, openSearch.cfg.Host)
}

// connecting and setting connection for opensearch
func (openSearch *openSearchConnector) Connect(ctx context.Context) error {
	signer, err := openSearch.openSearchSigner(ctx)
	if err != nil {
		return err
	}
	openSearch.logger.Debugf("Creating opensearch client %s", openSearch.connectionString())

	client, err := opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			MaxConnsPerHost: openSearch.cfg.MaxConnection,
		},
		Signer:        signer,
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
	if openSearch.cfg.Port > 0 {
		return fmt.Sprintf("ES %s://%s:%d", openSearch.cfg.Schema, openSearch.cfg.Host, openSearch.cfg.Port)
	}
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

func (sr *SearchResponse) Result(output interface{}) error {
	if sr.Error() != nil {
		return sr.Error()
	}
	if sr.Hits.Total.Value > 0 {
		err := mapstructure.Decode(sr.Hits.Hits, &output)
		if err != nil {
			return err
		}
	}
	return nil
}

// decoding the open search count response to given target struct ptr
func (sr *SearchResponseWithCount) Result(output interface{}) error {
	if sr.Error() != nil {
		return sr.Error()
	}
	if sr.Hits.Total > 0 {
		err := mapstructure.Decode(sr.Hits.Hits, &output)
		if err != nil {
			return err
		}
	}
	return nil
}

// return searchresponse from search query on given index and body with overall count
func (openSearch *openSearchConnector) SearchWithCount(ctx context.Context, index string, body string) *SearchResponseWithCount {
	searchResponse := &SearchResponseWithCount{}
	openSearch.logger.Debugf("searching with count query on index %s", index)
	err := openSearch.search(ctx, index, body, true, searchResponse)
	if err != nil {
		searchResponse.Err = err
		return searchResponse
	}
	openSearch.logger.Infof("returning opensearch `SearchWithCount` result with count %d time %v", searchResponse.Hits.Total, searchResponse.Took)
	return searchResponse
}

// return searchresponse from search query on given index and body
func (openSearch *openSearchConnector) Search(ctx context.Context, index string, body string) *SearchResponse {
	searchResponse := &SearchResponse{}
	openSearch.logger.Debugf("searching query on index %s", index)
	err := openSearch.search(ctx, index, body, false, searchResponse)
	if err != nil {
		searchResponse.Err = err
		return searchResponse
	}
	openSearch.logger.Infof("returning opensearch `Search` result with count %d time %v", searchResponse.Hits.Total.Value, searchResponse.Took)
	return searchResponse
}

// raw search execution for open search
func (openSearch *openSearchConnector) search(ctx context.Context, index string, body string, totalHitsAsInt bool, output interface{}) error {
	// only for benchmarking
	start := time.Now()

	openSearch.logger.Debugf("searching query started executing on index %s", index)
	searchQuery := opensearchapi.SearchRequest{
		Index:              []string{index},
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
		openSearch.logger.Errorf("error searching to opensearch status is not legal: %v", searchResponse.StatusCode)
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

// disconnect from opensearch client
func (c *openSearchConnector) Disconnect(ctx context.Context) error {

	// do somthing to close the connection
	// defer c.Connection.close()
	c.logger.Debug("Disconnecting with opensearch client.")
	c.Connection = nil
	return nil
}

// Wrapper for implimentation of "github.com/opensearch-project/opensearch-go/signer"
type Signer struct {
	Session awsSession.Session
	Service string
}

// signer implimentation
func (openSearch *openSearchConnector) openSearchSigner(ctx context.Context) (*Signer, error) {

	sessionOptions := awsSession.Options{
		Config:             aws.Config{Region: aws.String(openSearch.cfg.Auth.Region)},
		SharedConfigState:  awsSession.SharedConfigEnable,
		AssumeRoleDuration: time.Duration(float64(openSearch.cfg.TokenDuration) * float64(time.Minute)),
	}
	awsSession, err := awsSession.NewSessionWithOptions(sessionOptions)
	if err != nil {
		openSearch.logger.Errorf("failed to get session from given option %v due to %s", sessionOptions, err)
		return nil, fmt.Errorf("failed to get session from given option %v due to %s", sessionOptions, err)
	}
	return &Signer{
		Session: *awsSession,
		Service: "es",
	}, nil
}

// can find the reference "github.com/opensearch-project/opensearch-go/signer"
// an signer for request supported by opensearch implimentation
// similar implimentation as python-service-template but using aws-signer-v4 made it easier.
func (s Signer) SignRequest(req *http.Request) (err error) {
	signer := v4.NewSigner(s.Session.Config.Credentials)
	region := s.Session.Config.Region
	if region == nil || len(*region) == 0 {
		return fmt.Errorf("aws region cannot be empty")
	}
	if req.Body == nil {
		_, err = signer.Sign(req, nil, s.Service, *s.Session.Config.Region, time.Now().UTC())
		return
	}
	buf, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	_, err = signer.Sign(req, bytes.NewReader(buf), s.Service, *s.Session.Config.Region, time.Now().UTC())
	return
}

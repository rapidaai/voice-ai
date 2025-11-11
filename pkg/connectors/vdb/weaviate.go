package vdb_connectors

import (
	"context"
	"fmt"

	commons "github.com/rapidaai/pkg/commons"
	configs "github.com/rapidaai/pkg/configs"
	"github.com/rapidaai/pkg/connectors"
	"github.com/rapidaai/pkg/utils"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/grpc"
)

type WeaviateConnector interface {
	connectors.VectorConnector
	DB() *weaviate.Client
}

type weaviateConnector struct {
	cfg    *configs.WeaviateConfig
	logger commons.Logger
	db     *weaviate.Client
}

// TextSearch implements WeaviateConnector.
func (*weaviateConnector) TextSearch(ctx context.Context, collectionName string, query string, entities map[string]interface{}, opts *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	panic("unimplemented")
}

// VectorSearch implements WeaviateConnector.
func (*weaviateConnector) VectorSearch(ctx context.Context, collectionName string, queryVector []float64, entities map[string]interface{}, opts *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {
	panic("unimplemented")
}

// HybridSearch implements WeaviateConnector.
func (wv *weaviateConnector) HybridSearch(ctx context.Context,
	collectionName string,
	query string,
	queryVector []float64,
	entities map[string]interface{},
	opts *connectors.VectorSearchOptions) ([]map[string]interface{}, error) {

	gQL := wv.db.GraphQL().
		HybridArgumentBuilder().
		WithQuery(query).
		WithVector(utils.EmbeddingToFloat32(queryVector))
	builder := wv.db.GraphQL().Get()
	if opts != nil {
		if opts.TopK > 0 {
			builder.WithLimit(opts.TopK)
		}

		if opts.Alpha > 0 {
			gQL.WithAlpha(opts.Alpha)
		}
	}

	result, err := builder.WithHybrid(gQL).
		WithClassName(collectionName).Do(context.Background())
	if err != nil || len(result.Errors) > 0 {
		wv.logger.Errorf("unable to query vector db %v %+v", err, result.Errors)
		return nil, err
	}

	out := make([]map[string]interface{}, 0)
	for _, rt := range result.Data {
		out = append(out, rt.(map[string]interface{}))
	}
	return out, nil
}

func NewWeaviateConnector(config *configs.WeaviateConfig, logger commons.Logger) WeaviateConnector {
	return &weaviateConnector{cfg: config, logger: logger}
}

func (weaviateDB *weaviateConnector) DB() *weaviate.Client {
	return weaviateDB.db
}

// generating connection string from configuration
func (weaviateDB *weaviateConnector) connectionString() string {
	return fmt.Sprintf("%s://%s", weaviateDB.cfg.Scheme, weaviateDB.cfg.Host)
}

func (weaviateDB *weaviateConnector) Connect(ctx context.Context) error {
	weaviateDB.logger.Debugf("Creating weaviate client %s", weaviateDB.connectionString())
	cfg := weaviate.Config{
		Host:       weaviateDB.cfg.Host,
		Scheme:     weaviateDB.cfg.Scheme,
		AuthConfig: auth.ApiKey{Value: weaviateDB.cfg.Auth.ApiKey},
		GrpcConfig: &grpc.Config{
			Secured: true,
			Host:    weaviateDB.cfg.Host,
		},
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		weaviateDB.logger.Errorf("connecting to weaviate db with ends with error %v", err)
		return err
	}

	weaviateDB.db = client
	return nil
}
func (weaviateDB *weaviateConnector) Name() string {
	return "weaviate"
}
func (weaviateDB *weaviateConnector) IsConnected(ctx context.Context) bool {
	_, err := weaviateDB.db.Schema().Getter().Do(context.Background())
	if err != nil {
		weaviateDB.logger.Errorf("connecting to weaviate db with ends with error %v", err)
		return false
	}
	return true
}
func (weaviateDB *weaviateConnector) Disconnect(ctx context.Context) error {
	weaviateDB.db = nil
	return nil
}

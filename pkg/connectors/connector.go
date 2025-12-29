// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package connectors

import (
	"context"
	"encoding/json"
	"fmt"
)

// Connector ideated from python-service-template
// An interface which provide common behavior for all the data source connectors.
type Connector interface {
	Connect(ctx context.Context) error
	Name() string
	IsConnected(ctx context.Context) bool
	Disconnect(ctx context.Context) error
}

type VectorSearchOptions struct {
	// fusion for hybrid search
	Alpha float32

	// limit
	TopK int

	// max score that will match
	MinScore float32

	// list of string needed
	Source []string
}

func (vso *VectorSearchOptions) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(vso)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VectorSearchOptions: %w", err)
	}
	return string(jsonBytes), nil
}

func NewDefaultVectorSearchOptions(ots ...SearchOptions) *VectorSearchOptions {
	so := &VectorSearchOptions{
		TopK:     5,
		MinScore: 0.5,
		Source:   []string{"text", "metadata"},
	}
	return so.WithOptions(ots...)

}

func (opts *VectorSearchOptions) WithOptions(options ...SearchOptions) *VectorSearchOptions {
	for _, opt := range options {
		opt(opts)
	}
	return opts
}

// type ChatOptions = RequestOption
type SearchOptions func(*VectorSearchOptions)

func WithTopK(topK int) SearchOptions {
	return func(cc *VectorSearchOptions) {
		cc.TopK = topK
	}
}
func WithAlpha(alpha float32) SearchOptions {
	return func(cc *VectorSearchOptions) {
		cc.Alpha = alpha
	}
}
func WithMinScore(min float32) SearchOptions {
	return func(cc *VectorSearchOptions) {
		cc.MinScore = min
	}
}

func WithSource(attr []string) SearchOptions {
	return func(cc *VectorSearchOptions) {
		cc.Source = attr
	}
}

type VectorConnector interface {
	Connector
	VectorSearch(ctx context.Context,
		collectionName string,
		queryVector []float64,
		filter map[string]interface{},
		opts *VectorSearchOptions) ([]map[string]interface{}, error)
	HybridSearch(ctx context.Context,
		collectionName string,
		query string,
		queryVector []float64,
		filter map[string]interface{},
		opts *VectorSearchOptions) ([]map[string]interface{}, error)
	TextSearch(ctx context.Context,
		collectionName string,
		query string,
		filter map[string]interface{},
		opts *VectorSearchOptions) ([]map[string]interface{}, error)
}

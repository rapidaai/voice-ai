package plugins_types

import (
	"context"

	web_client "github.com/rapidaai/pkg/clients/web"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/types"
)

const (
	StatusSuccess = "SUCCESS"
	StatusFail    = "FAIL"
)

type ExecuteDeps struct {
	VaultClient web_client.VaultClient
	Logger      commons.Logger
	Auth        *types.Authentication
}

type ExecuteRequest struct {
	Operation string
	Provider  string
	Input     map[string]interface{}
	Config    map[string]interface{}
}

type Result struct {
	Status    string                 `json:"status"`
	Provider  string                 `json:"provider,omitempty"`
	Operation string                 `json:"operation,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

func Success(provider, operation string, data map[string]interface{}) *Result {
	if data == nil {
		data = map[string]interface{}{}
	}
	return &Result{
		Status:    StatusSuccess,
		Provider:  provider,
		Operation: operation,
		Data:      data,
	}
}

func Failure(provider, operation, errMsg string, data map[string]interface{}) *Result {
	if data == nil {
		data = map[string]interface{}{}
	}
	return &Result{
		Status:    StatusFail,
		Provider:  provider,
		Operation: operation,
		Error:     errMsg,
		Data:      data,
	}
}

func (r *Result) ToMap() map[string]interface{} {
	if r == nil {
		return map[string]interface{}{
			"status": StatusFail,
			"error":  "nil plugin result",
		}
	}
	m := map[string]interface{}{
		"status":    r.Status,
		"provider":  r.Provider,
		"operation": r.Operation,
	}
	if len(r.Data) > 0 {
		m["data"] = r.Data
	}
	if r.Error != "" {
		m["error"] = r.Error
	}
	return m
}

type Provider interface {
	Code() string
	Execute(ctx context.Context, operation string, input map[string]interface{}, credential map[string]interface{}, logger commons.Logger) (map[string]interface{}, error)
}

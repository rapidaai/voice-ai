package clients_pogos

import (
	"fmt"

	"github.com/lexatic/web-backend/pkg/types"
	integration_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	provider_api "github.com/lexatic/web-backend/protos/lexatic-backend"
	vault_api "github.com/lexatic/web-backend/protos/lexatic-backend"
)

// generating audit information for the request
func GenerateAuditInfo[T any](request *RequestData[T]) *integration_api.AuditInfo {
	existing := request.Metadata
	existing["provider_model_id"] = fmt.Sprintf("%d", request.ProviderModelId)
	existing["model_name"] = request.ProviderModelName
	existing["provider_name"] = request.ProviderName
	existing["vault_id"] = fmt.Sprintf("%d", request.Credential.Id)
	existing["vault_name"] = request.Credential.Name

	// additionalData, err := structpb.NewStruct(existing)
	// if err != nil {
	// 	fmt.Printf("unable to create struct out of metdata %v", err)
	// }
	return &integration_api.AuditInfo{
		OrganizationId: request.OrganizationId,
		ProjectId:      request.ProjectId,
		AdditionalData: existing,
	}
}

type ProviderModelParameter struct {
	Id                      uint64
	Name                    string
	Value                   string
	Type                    string
	ProviderModelVariableId uint64
}

type Interaction struct {
	RequestId     uint64
	ResponseId    uint64
	Status        string
	RequestPrompt string
	RequestRole   string
	Response      string
	ResponseRole  string
	CreatedBy     uint64
}

type RequestData[T any] struct {
	// provider information
	ProviderId   uint64
	ProviderName string

	// model information
	ProviderModelId         uint64
	ProviderModelName       string
	ProviderModelParameters []*ProviderModelParameter
	Version                 string

	// either it will be string or conversaction
	GlobalPrompt T
	SystemPrompt *string

	// credential
	Credential *vault_api.ProviderCredential

	// audit information
	OrganizationId uint64
	ProjectId      uint64

	Metadata map[string]string
}

func GenerateModelParameter(params []*ProviderModelParameter) []*integration_api.ModelParameter {
	parameters := make([]*integration_api.ModelParameter, 0)
	for _, prm := range params {
		parameters = append(parameters, &integration_api.ModelParameter{
			Key:   prm.Name,
			Value: prm.Value,
			Type:  prm.Type,
		})
	}
	return parameters
}

func ComposePromptModelData[T string | []*Interaction](mldR *provider_api.Model,
	credential *vault_api.ProviderCredential,
	globalPrompt T, systemPrompt *string,
	parameters interface{},
	projectId, organizationId uint64,
	extraArgs map[string]string) *RequestData[T] {

	argument := &RequestData[T]{
		ProviderId:   mldR.ProviderId,
		ProviderName: mldR.Provider.Name,

		// model information
		ProviderModelId:         mldR.Id,
		ProviderModelName:       mldR.Name,
		ProviderModelParameters: CastToParameters(parameters),
		Version:                 mldR.Version,
		// either it will be string or conversaction
		GlobalPrompt: globalPrompt,
		SystemPrompt: systemPrompt,
		Credential:   credential,
		// audit information
		OrganizationId: organizationId,
		ProjectId:      projectId,
		Metadata:       extraArgs,
	}
	return argument
}

func CastToParameters(in interface{}) []*ProviderModelParameter {
	out := make([]*ProviderModelParameter, 0)
	err := types.Cast(in, &out)
	if err != nil {
		fmt.Printf("illegal params %v", err)
	}
	return out
}

func ToConversaction(interactions []*Interaction) []*integration_api.Conversation {
	conversations := make([]*integration_api.Conversation, 0)
	for _, c := range interactions {
		conversations = append(conversations, &integration_api.Conversation{
			Role:    c.RequestRole,
			Content: c.RequestPrompt,
		})
		if c.Response != "" && c.Status == "SUCCESS" {
			conversations = append(conversations, &integration_api.Conversation{
				Role:    c.ResponseRole,
				Content: c.Response,
			})
		}
	}
	return conversations
}

package assistant_api

import (
	"testing"

	"github.com/stretchr/testify/require"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	gorm_model "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
)

func TestAssistantConfigurationOpenAPIIncludesAuditActors(t *testing.T) {
	configuration := &internal_assistant_entity.AssistantConfiguration{
		Audited: gorm_model.Audited{Id: 17},
		Mutable: gorm_model.Mutable{ActorAudit: gorm_model.ActorAudit{
			CreatedActor: &types.ActorIdentity{Type: types.ActorTypeService, ID: 41},
			UpdatedActor: &types.ActorIdentity{Type: types.ActorTypeSystem, ID: 42},
		}},
	}

	result := assistantConfigurationOpenAPI(configuration)

	require.NotNil(t, result.CreatedActor)
	require.Equal(t, "service", string(result.CreatedActor.Type))
	require.Equal(t, "41", string(result.CreatedActor.Id))
	require.NotNil(t, result.UpdatedActor)
	require.Equal(t, "system", string(result.UpdatedActor.Type))
	require.Equal(t, "42", string(result.UpdatedActor.Id))
}

func TestAssistantConfigurationOpenAPIOmitsInvalidAuditActor(t *testing.T) {
	configuration := &internal_assistant_entity.AssistantConfiguration{
		Mutable: gorm_model.Mutable{ActorAudit: gorm_model.ActorAudit{
			CreatedActor: &types.ActorIdentity{Type: types.ActorTypeUnknown},
		}},
	}

	result := assistantConfigurationOpenAPI(configuration)

	require.Nil(t, result.CreatedActor)
}

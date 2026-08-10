package adapter_internal

import (
	"testing"

	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	"github.com/rapidaai/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestDeploymentBehavior_AppliesConversationExperienceOverrides(t *testing.T) {
	defaultGreeting := "saved greeting"
	defaultGreetingInterruptible := true
	defaultMistake := "saved mistake"
	defaultUnclearInputTimeout := 0.5
	defaultUnclearInputMessage := "saved unclear"
	defaultIdleTimeout := uint64(10)
	defaultIdleTimeoutBackoff := uint64(2)
	defaultIdleTimeoutMessage := "saved idle"
	defaultMaxSessionDuration := uint64(300)

	requestor := &genericRequestor{
		source: utils.PhoneCall,
		assistant: &internal_assistant_entity.Assistant{
			AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					Greeting:              &defaultGreeting,
					GreetingInterruptible: &defaultGreetingInterruptible,
					Mistake:               &defaultMistake,
					UnclearInputTimeout:   &defaultUnclearInputTimeout,
					UnclearInputMessage:   &defaultUnclearInputMessage,
					IdleTimeout:           &defaultIdleTimeout,
					IdleTimeoutBackoff:    &defaultIdleTimeoutBackoff,
					IdleTimeoutMessage:    &defaultIdleTimeoutMessage,
					MaxSessionDuration:    &defaultMaxSessionDuration,
				},
			},
		},
		options: map[string]interface{}{
			internal_options.ExperienceOptionGreeting:              "override greeting",
			internal_options.ExperienceOptionGreetingInterruptible: false,
			internal_options.ExperienceOptionMistake:               "override mistake",
			internal_options.ExperienceOptionUnclearInputTimeout:   "1.7",
			internal_options.ExperienceOptionUnclearInputMessage:   "override unclear",
			internal_options.ExperienceOptionIdleTimeout:           float64(15),
			internal_options.ExperienceOptionIdleTimeoutBackoff:    "4",
			internal_options.ExperienceOptionIdleTimeoutMessage:    "override idle",
			internal_options.ExperienceOptionMaxSessionDuration:    uint64(600),
		},
	}

	behavior, err := requestor.deploymentBehavior()
	require.NoError(t, err)
	require.NotNil(t, behavior)
	require.Equal(t, "override greeting", *behavior.Greeting)
	require.False(t, *behavior.GreetingInterruptible)
	require.Equal(t, "override mistake", *behavior.Mistake)
	require.Equal(t, 1.7, *behavior.UnclearInputTimeout)
	require.Equal(t, "override unclear", *behavior.UnclearInputMessage)
	require.Equal(t, uint64(15), *behavior.IdleTimeout)
	require.Equal(t, uint64(4), *behavior.IdleTimeoutBackoff)
	require.Equal(t, "override idle", *behavior.IdleTimeoutMessage)
	require.Equal(t, uint64(600), *behavior.MaxSessionDuration)

	saved := requestor.assistant.AssistantPhoneDeployment.AssistantDeploymentBehavior
	require.Equal(t, defaultGreeting, *saved.Greeting)
	require.True(t, *saved.GreetingInterruptible)
	require.Equal(t, defaultMistake, *saved.Mistake)
	require.Equal(t, defaultUnclearInputTimeout, *saved.UnclearInputTimeout)
	require.Equal(t, defaultUnclearInputMessage, *saved.UnclearInputMessage)
	require.Equal(t, defaultIdleTimeout, *saved.IdleTimeout)
	require.Equal(t, defaultIdleTimeoutBackoff, *saved.IdleTimeoutBackoff)
	require.Equal(t, defaultIdleTimeoutMessage, *saved.IdleTimeoutMessage)
	require.Equal(t, defaultMaxSessionDuration, *saved.MaxSessionDuration)
}

func TestDeploymentBehavior_IgnoresInvalidExperienceOverrideValue(t *testing.T) {
	defaultIdleTimeout := uint64(10)
	requestor := &genericRequestor{
		source: utils.PhoneCall,
		assistant: &internal_assistant_entity.Assistant{
			AssistantPhoneDeployment: &internal_assistant_entity.AssistantPhoneDeployment{
				AssistantDeploymentBehavior: internal_assistant_entity.AssistantDeploymentBehavior{
					IdleTimeout: &defaultIdleTimeout,
				},
			},
		},
		options: map[string]interface{}{
			internal_options.ExperienceOptionIdleTimeout: -1,
		},
	}

	behavior, err := requestor.deploymentBehavior()
	require.NoError(t, err)
	require.NotNil(t, behavior)
	require.Equal(t, defaultIdleTimeout, *behavior.IdleTimeout)
}

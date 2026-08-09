// Copyright (c) 2023-2025 RapidaAI
// Author: Prashant Srivastav <prashant@rapida.ai>
//
// Licensed under GPL-2.0 with Rapida Additional Terms.
// See LICENSE.md or contact sales@rapida.ai for commercial usage.
package adapter_internal

import (
	internal_assistant_entity "github.com/rapidaai/api/assistant-api/internal/entity/assistants"
	internal_options "github.com/rapidaai/api/assistant-api/internal/options"
	"github.com/rapidaai/pkg/utils"
)

func (r *genericRequestor) deploymentBehavior() (*internal_assistant_entity.AssistantDeploymentBehavior, error) {
	assistant, err := r.Assistant()
	if err != nil {
		return nil, err
	}

	switch r.source {
	case utils.PhoneCall:
		if assistant.AssistantPhoneDeployment != nil {
			return r.withExperienceOverrides(&assistant.AssistantPhoneDeployment.AssistantDeploymentBehavior), nil
		}
	case utils.Whatsapp:
		if assistant.AssistantWhatsappDeployment != nil {
			return r.withExperienceOverrides(&assistant.AssistantWhatsappDeployment.AssistantDeploymentBehavior), nil
		}
	case utils.SDK:
		if assistant.AssistantApiDeployment != nil {
			return r.withExperienceOverrides(&assistant.AssistantApiDeployment.AssistantDeploymentBehavior), nil
		}
	case utils.WebPlugin:
		if assistant.AssistantWebPluginDeployment != nil {
			return r.withExperienceOverrides(&assistant.AssistantWebPluginDeployment.AssistantDeploymentBehavior), nil
		}
	case utils.Debugger:
		if assistant.AssistantDebuggerDeployment != nil {
			return r.withExperienceOverrides(&assistant.AssistantDebuggerDeployment.AssistantDeploymentBehavior), nil
		}
	}

	return nil, errDeploymentNotEnabled
}

func (r *genericRequestor) withExperienceOverrides(
	behavior *internal_assistant_entity.AssistantDeploymentBehavior,
) *internal_assistant_entity.AssistantDeploymentBehavior {
	if behavior == nil {
		return nil
	}

	resolved := *behavior
	opts := r.GetOptions()
	if len(opts) == 0 {
		return &resolved
	}

	if value, err := opts.GetString(internal_options.ExperienceOptionGreeting); err == nil {
		resolved.Greeting = &value
	}
	if value, err := opts.GetBool(internal_options.ExperienceOptionGreetingInterruptible); err == nil {
		resolved.GreetingInterruptible = &value
	}
	if value, err := opts.GetString(internal_options.ExperienceOptionMistake); err == nil {
		resolved.Mistake = &value
	}
	if value, err := opts.GetFloat64(internal_options.ExperienceOptionUnclearInputTimeout); err == nil {
		resolved.UnclearInputTimeout = &value
	}
	if value, err := opts.GetString(internal_options.ExperienceOptionUnclearInputMessage); err == nil {
		resolved.UnclearInputMessage = &value
	}
	if value, err := opts.GetUint64(internal_options.ExperienceOptionIdleTimeout); err == nil {
		resolved.IdleTimeout = &value
	}
	if value, err := opts.GetUint64(internal_options.ExperienceOptionIdleTimeoutBackoff); err == nil {
		resolved.IdleTimeoutBackoff = &value
	}
	if value, err := opts.GetString(internal_options.ExperienceOptionIdleTimeoutMessage); err == nil {
		resolved.IdleTimeoutMessage = &value
	}
	if value, err := opts.GetUint64(internal_options.ExperienceOptionMaxSessionDuration); err == nil {
		resolved.MaxSessionDuration = &value
	}

	return &resolved
}

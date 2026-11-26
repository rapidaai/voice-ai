import { useEffect, useState } from 'react';
import {
  AgentflowDefinition,
  AgentflowVersionBuilder,
} from '@/app/pages/assistant/actions/create-assistant-version/agentflow-version-builder';
import { ErrorContainer } from '@/app/components/ui/feedback/error-container';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { CreateAssistantProviderRequest } from '@rapidaai/react';
import { Struct } from 'google-protobuf/google/protobuf/struct_pb';
import type { JavaScriptValue } from 'google-protobuf/google/protobuf/struct_pb';
import { useParams } from 'react-router-dom';
import toast from 'react-hot-toast/headless';
import {
  createAssistantProviderFromRequest,
  getAssistantByIdWithApi,
} from '@/clients/assistant.client';

const toAgentflowStruct = (definition: AgentflowDefinition) =>
  Struct.fromJavaScript(
    JSON.parse(JSON.stringify(definition)) as Record<string, JavaScriptValue>,
  );

const isAgentflowDefinition = (
  definition: unknown,
): definition is AgentflowDefinition => {
  if (!definition || typeof definition !== 'object') return false;
  const candidate = definition as Partial<AgentflowDefinition>;
  return (
    typeof candidate.schemaVersion === 'string' &&
    typeof candidate.entryNodeId === 'string' &&
    Array.isArray(candidate.nodes) &&
    Array.isArray(candidate.edges)
  );
};

export function CreateAgentflowVersion() {
  const { assistantId } = useParams();
  const navigator = useGlobalNavigation();

  if (!assistantId) {
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={navigator.goToAssistantListing}
          code="403"
          actionLabel="Back to assistants"
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );
  }

  return <CreateAgentflowVersionBuilder assistantId={assistantId} />;
}

function CreateAgentflowVersionBuilder({
  assistantId,
}: {
  assistantId: string;
}) {
  const { authId, token, projectId } = useCurrentCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const navigator = useGlobalNavigation();
  const [initialDefinition, setInitialDefinition] =
    useState<AgentflowDefinition>();

  useEffect(() => {
    let isMounted = true;
    const loadCurrentAgentflow = async () => {
      showLoader('overlay');
      try {
        const response = await getAssistantByIdWithApi({
          assistantId,
          auth: { projectId, token, userId: authId },
        });

        if (!isMounted) return;

        if (!response?.getSuccess()) {
          toast.error(
            response?.getError()?.getHumanmessage() ??
              'Unable to load current agentflow.',
          );
          return;
        }

        const definition =
          response
            .getData()
            ?.getAssistantprovideragentflow()
            ?.getDefinition()
            ?.toJavaScript() ?? null;

        if (isAgentflowDefinition(definition)) {
          setInitialDefinition(definition);
        }
      } catch {
        if (isMounted) {
          toast.error('Unable to load current agentflow.');
        }
      } finally {
        if (isMounted) {
          hideLoader();
        }
      }
    };

    loadCurrentAgentflow();

    return () => {
      isMounted = false;
    };
  }, [assistantId, authId, hideLoader, projectId, showLoader, token]);

  const saveAgentflowVersion = async (definition: AgentflowDefinition) => {
    showLoader('overlay');
    try {
      const request = new CreateAssistantProviderRequest();
      const agentflow =
        new CreateAssistantProviderRequest.CreateAssistantProviderAgentflow();
      agentflow.setSchemaversion(definition.schemaVersion);
      agentflow.setDefinition(toAgentflowStruct(definition));

      request.setAssistantid(assistantId);
      request.setDescription(definition.description ?? '');
      request.setAgentflow(agentflow);

      const response = await createAssistantProviderFromRequest({
        request,
        auth: { projectId, token, userId: authId },
      });

      if (!response?.getSuccess()) {
        throw new Error(
          response?.getError()?.getHumanmessage() ??
            'Unable to create agentflow version.',
        );
      }

      navigator.goToAssistantVersions(assistantId);
    } finally {
      hideLoader();
    }
  };

  return (
    <AgentflowVersionBuilder
      title="Create Agentflow Version"
      initialDefinition={initialDefinition}
      onSave={saveAgentflowVersion}
    />
  );
}

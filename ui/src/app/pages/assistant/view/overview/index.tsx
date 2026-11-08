import { Assistant } from '@rapidaai/react';
import { Checklist } from '@carbon/ibm-products/lib/components/Checklist/Checklist';
import type {
  Kind as ChecklistTaskKind,
  Theme as ChecklistTheme,
} from '@carbon/ibm-products/lib/components/Checklist/Checklist.types';
import { SectionLoader } from '@/app/components/ui/feedback';
import { AssistantAnalytics } from '@/app/pages/assistant/view/overview/assistant-analytics';
import { useRapidaStore } from '@/stores/app';
import { FC } from 'react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { SourceControl } from '@carbon/icons-react';
import { ResourceDetailHeader } from '@/app/components/layout/resource-detail-header';
import { useTheme } from '@/theme/theme-provider';

const uncheckedChecklistTask = 'unchecked' as ChecklistTaskKind;
const checkedChecklistTask = 'checked' as ChecklistTaskKind;
const indeterminateChecklistTask = 'indeterminate' as ChecklistTaskKind;

export const Overview: FC<{ currentAssistant: Assistant }> = ({
  currentAssistant,
}) => {
  const rapidaContext = useRapidaStore();
  const navigation = useGlobalNavigation();
  const { resolvedMode } = useTheme();
  const assistantId = currentAssistant.getId();
  const deployments = [
    currentAssistant.getApideployment(),
    currentAssistant.getDebuggerdeployment(),
    currentAssistant.getWebplugindeployment(),
    currentAssistant.getPhonedeployment(),
  ].filter(Boolean);
  const hasConfiguredDeployment = deployments.length > 0;
  const hasActiveDeployment = deployments.some(
    deployment => deployment?.getStatus().toUpperCase() === 'ACTIVE',
  );
  const deploymentProgress = hasConfiguredDeployment ? 75 : 25;

  if (rapidaContext.loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="flex flex-col flex-1 grow">
      <ResourceDetailHeader
        collectionHref="/deployment/assistant"
        collectionLabel="Assistants"
        resourceName={currentAssistant.getName()}
        actionsLabel="Assistant overview header actions"
        primaryAction={{
          label: 'Create new version',
          icon: SourceControl,
          onClick: () => navigation.goToCreateAssistantVersion(assistantId),
        }}
      />

      {!hasActiveDeployment && (
        <Checklist
          chartLabel={`${deploymentProgress}% complete`}
          chartValue={deploymentProgress / 100}
          checklistAriaLabel="Assistant deployment checklist"
          checklistToggleAriaLabel="Toggle assistant deployment checklist"
          className="app-onboarding-checklist fixed bottom-4 right-4 z-30 max-h-[calc(100vh-2rem)] w-[calc(100%_-_5rem)] max-w-[400px] overflow-auto border border-border-strong shadow-xl"
          onClickViewAll={() => navigation.goToDeploymentAssistant(assistantId)}
          taskLists={[
            {
              title: 'Deployment setup',
              tasks: [
                {
                  kind: checkedChecklistTask,
                  label: 'Assistant created',
                },
                {
                  kind: hasConfiguredDeployment
                    ? checkedChecklistTask
                    : uncheckedChecklistTask,
                  label: 'Choose a deployment channel',
                  onClick: () =>
                    navigation.goToDeploymentAssistant(assistantId),
                },
                {
                  kind: hasConfiguredDeployment
                    ? checkedChecklistTask
                    : uncheckedChecklistTask,
                  label: 'Configure deployment settings',
                  onClick: () =>
                    navigation.goToDeploymentAssistant(assistantId),
                },
                {
                  kind: hasConfiguredDeployment
                    ? indeterminateChecklistTask
                    : uncheckedChecklistTask,
                  label: 'Enable your deployment',
                  onClick: () =>
                    navigation.goToDeploymentAssistant(assistantId),
                },
              ],
            },
          ]}
          title="Deploy your assistant"
          theme={resolvedMode as ChecklistTheme}
          toggleLabel="Collapse or expand deployment setup"
          viewAllLabel="View deployment options"
        />
      )}

      <AssistantAnalytics assistant={currentAssistant} />
    </div>
  );
};

import { Assistant } from '@rapidaai/react';
import { SectionLoader } from '@/app/components/ui/feedback';
import { AssistantAnalytics } from '@/app/pages/assistant/view/overview/assistant-analytics';
import { useRapidaStore } from '@/stores/app';
import { FC } from 'react';
import { ActionNotification } from '@/app/components/ui/feedback';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { SourceControl } from '@carbon/icons-react';
import { ResourceDetailHeader } from '@/app/components/layout/resource-detail-header';

export const Overview: FC<{ currentAssistant: Assistant }> = ({
  currentAssistant,
}) => {
  const rapidaContext = useRapidaStore();
  const navigation = useGlobalNavigation();

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
          onClick: () =>
            navigation.goToCreateAssistantVersion(currentAssistant.getId()),
        }}
      />

      {/* ── Notifications ── */}
      {!currentAssistant.getApideployment() &&
        !currentAssistant.getDebuggerdeployment() &&
        !currentAssistant.getWebplugindeployment() &&
        !currentAssistant.getPhonedeployment() && (
          <div className="max-md:bg-notification-warning">
            <ActionNotification
              kind="warning"
              title="Your assistant is ready, but not live yet."
              subtitle="It looks like your assistant isn't deployed to any channel."
              actionButtonLabel="Enable deployment"
              onActionButtonClick={() =>
                navigation.goToDeploymentAssistant(currentAssistant.getId())
              }
            />
          </div>
        )}

      {/* ── Dashboard content ── */}
      <AssistantAnalytics assistant={currentAssistant} />
    </div>
  );
};

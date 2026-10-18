import { Assistant } from '@rapidaai/react';
import { SectionLoader } from '@/app/components/ui/feedback';
import { AssistantAnalytics } from '@/app/pages/assistant/view/overview/assistant-analytics';
import { useRapidaStore } from '@/stores/app';
import { FC } from 'react';
import { ActionNotification } from '@/app/components/ui/feedback';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  Breadcrumb,
  BreadcrumbItem,
  Button,
  HeaderGlobalBar,
} from '@carbon/react';
import { SourceControl } from '@carbon/icons-react';

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
      <header className="flex h-12 shrink-0 items-center justify-between bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-800">
        <div className="min-w-0 pl-4">
          <Breadcrumb noTrailingSlash>
            <BreadcrumbItem href="/deployment/assistant">
              Assistants
            </BreadcrumbItem>
            <BreadcrumbItem isCurrentPage>
              <span className="block max-w-[42vw] truncate">
                {currentAssistant.getName()}
              </span>
            </BreadcrumbItem>
          </Breadcrumb>
        </div>
        <HeaderGlobalBar
          aria-label="Assistant overview header actions"
          className="h-full items-center"
        >
          <Button
            aria-label="Create new version"
            kind="primary"
            size="lg"
            renderIcon={SourceControl}
            className="h-full! min-h-full! items-center justify-center whitespace-nowrap"
            onClick={() =>
              navigation.goToCreateAssistantVersion(currentAssistant.getId())
            }
          >
            Create new version
          </Button>
        </HeaderGlobalBar>
      </header>

      {/* ── Notifications ── */}
      {!currentAssistant.getApideployment() &&
        !currentAssistant.getDebuggerdeployment() &&
        !currentAssistant.getWebplugindeployment() &&
        !currentAssistant.getPhonedeployment() && (
          <div className="max-md:bg-[var(--cds-notification-background-warning)]">
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

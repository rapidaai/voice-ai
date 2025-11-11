import { Conversations } from '@/app/pages/assistant/view/conversations';
import { Overview } from '@/app/pages/assistant/view/overview';
import { Version } from '@/app/pages/assistant/view/version-list';
import { useAssistantPageStore } from '@/hooks/use-assistant-page-store';
import { useParams } from 'react-router-dom';

export const ViewAssistantPage = () => {
  const assistantAction = useAssistantPageStore();
  const { tab = 'overview' } = useParams();
  if (!assistantAction.currentAssistant) {
    return null;
  }
  const renderTabContent = () => {
    if (tab === 'overview') {
      return <Overview currentAssistant={assistantAction.currentAssistant!} />;
    } else if (tab === 'version-history') {
      return (
        <Version
          assistant={assistantAction.currentAssistant!}
          onReload={() => {}}
        />
      );
    } else if (tab === 'sessions') {
      return (
        <Conversations currentAssistant={assistantAction.currentAssistant!} />
      );
    } else {
      return <Overview currentAssistant={assistantAction.currentAssistant!} />;
    }
  };

  return <>{renderTabContent()}</>;
};

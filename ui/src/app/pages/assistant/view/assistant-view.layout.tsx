import { Helmet } from '@/app/components/app-shell/helmet';
import { useRapidaStore } from '@/stores/app';
import { useCredential } from '@/hooks/use-credential';
import { FC, HTMLAttributes, useEffect, useState } from 'react';
import { Outlet, useParams } from 'react-router-dom';
import { cn } from '@/utils';
import { useAssistantPageStore } from '@/stores/assistant/assistant.store';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { ErrorContainer } from '@/app/components/ui/feedback';
import { getAssistantById } from '@/clients/assistant.client';
import { AssistantSideNav } from './assistant-side-nav';

export const AssistantViewLayout: FC<HTMLAttributes<HTMLDivElement>> = () => {
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const currentAssistant = useAssistantPageStore(
    state => state.currentAssistant,
  );
  const clearAssistantState = useAssistantPageStore(state => state.clear);
  const setCurrentAssistant = useAssistantPageStore(
    state => state.onChangeCurrentAssistant,
  );
  const { assistantId } = useParams();
  const [navExpanded, setNavExpanded] = useState(
    () => !window.matchMedia?.('(max-width: 767px)').matches,
  );

  const { goToAssistantListing } = useGlobalNavigation();

  const [unknownState, setUnknownState] = useState(false);

  useEffect(() => {
    const media = window.matchMedia?.('(max-width: 767px)');
    if (!media) return undefined;

    const collapseOnMobile = (event: MediaQueryListEvent) => {
      if (event.matches) setNavExpanded(false);
    };

    media.addEventListener('change', collapseOnMobile);
    return () => media.removeEventListener('change', collapseOnMobile);
  }, []);

  useEffect(() => {
    clearAssistantState();
    setUnknownState(false);
    if (assistantId) {
      showLoader();
      getAssistantById({
        assistantId,
        auth: { projectId, token, userId },
      })
        .then(epmr => {
          hideLoader();
          if (epmr?.getSuccess() && epmr.getData()) {
            setCurrentAssistant(epmr.getData()!);
            return;
          }

          setUnknownState(true);
        })
        .catch(() => {
          setUnknownState(true);
          hideLoader();
        });
    }
  }, [
    assistantId,
    clearAssistantState,
    hideLoader,
    projectId,
    setCurrentAssistant,
    showLoader,
    token,
    userId,
  ]);

  if (unknownState)
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={goToAssistantListing}
          code="403"
          actionLabel="Back to assistants"
          actionIcon={null}
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );

  return (
    <div className={cn('flex h-full flex-1 overflow-hidden')}>
      <Helmet title="Hosted Assistant" />

      {/* ── Left side nav (config-driven) ── */}
      {assistantId && (
        <AssistantSideNav
          assistantId={assistantId}
          assistant={currentAssistant}
          expanded={navExpanded}
          onToggle={() => setNavExpanded(expanded => !expanded)}
        />
      )}

      {/* ── Main content area ── */}
      <div className="flex flex-col flex-1 overflow-auto">
        <Outlet />
      </div>
    </div>
  );
};

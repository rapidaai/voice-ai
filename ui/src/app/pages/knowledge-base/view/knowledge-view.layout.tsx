import { useCallback, useEffect } from 'react';
import { Outlet, useParams } from 'react-router-dom';
import { getKnowledgeBaseDetail } from '@/clients/knowledge.client';
import { ErrorContainer } from '@/app/components/ui/feedback';
import { useCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useRapidaStore } from '@/stores/app';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';

export interface KnowledgeViewOutletContext {
  onReload: () => void;
}

export function KnowledgeViewLayout() {
  const { id } = useParams();
  const [userId, token, projectId] = useCredential();
  const { showLoader, hideLoader } = useRapidaStore();
  const { goTo } = useGlobalNavigation();
  const status = useKnowledgePageStore(state => state.currentKnowledgeStatus);
  const clearCurrentKnowledge = useKnowledgePageStore(
    state => state.onClearCurrentKnowledge,
  );
  const setCurrentKnowledge = useKnowledgePageStore(
    state => state.onChangeCurrentKnowledge,
  );
  const setCurrentKnowledgeStatus = useKnowledgePageStore(
    state => state.onChangeCurrentKnowledgeStatus,
  );

  const onReload = useCallback(() => {
    clearCurrentKnowledge();
    if (!id) {
      setCurrentKnowledgeStatus('error');
      return;
    }

    setCurrentKnowledgeStatus('loading');
    showLoader('overlay');
    getKnowledgeBaseDetail({
      knowledgeId: id,
      auth: { projectId, token, userId },
      callback: (_error, response) => {
        hideLoader();
        const knowledge = response?.getSuccess() ? response.getData() : null;
        if (knowledge) {
          setCurrentKnowledge(knowledge);
          return;
        }
        setCurrentKnowledgeStatus('error');
      },
    });
  }, [
    clearCurrentKnowledge,
    hideLoader,
    id,
    projectId,
    setCurrentKnowledge,
    setCurrentKnowledgeStatus,
    showLoader,
    token,
    userId,
  ]);

  useEffect(() => {
    onReload();
  }, [onReload]);

  if (status === 'error') {
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={() => goTo('/knowledge')}
          code="404"
          actionLabel="Back to knowledge"
          actionIcon={null}
          title="Knowledge not available"
          description="This knowledge base may have been removed or you may not have access to it."
        />
      </div>
    );
  }

  return <Outlet context={{ onReload }} />;
}

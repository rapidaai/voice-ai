import { useCallback, useEffect, useRef } from 'react';
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

interface LoadedKnowledgeContext {
  knowledgeId: string;
  projectId: string;
  token: string;
  userId: string;
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
  const requestRef = useRef(0);
  const pendingRequestRef = useRef<number | null>(null);
  const loadedContextRef = useRef<LoadedKnowledgeContext | null>(null);

  const onReload = useCallback(() => {
    const requestId = ++requestRef.current;
    const requestContext = id
      ? { knowledgeId: id, projectId, token, userId }
      : null;
    const loadedContext = loadedContextRef.current;
    const hasCurrentKnowledge =
      requestContext !== null &&
      loadedContext?.knowledgeId === requestContext.knowledgeId &&
      loadedContext.projectId === requestContext.projectId &&
      loadedContext.token === requestContext.token &&
      loadedContext.userId === requestContext.userId &&
      useKnowledgePageStore.getState().currentKnowledge?.getId() ===
        requestContext.knowledgeId;
    if (!hasCurrentKnowledge) {
      loadedContextRef.current = null;
      clearCurrentKnowledge();
    }
    if (!requestContext) {
      setCurrentKnowledgeStatus('error');
      return;
    }

    setCurrentKnowledgeStatus('loading');
    showLoader('overlay');
    pendingRequestRef.current = requestId;
    getKnowledgeBaseDetail({
      knowledgeId: requestContext.knowledgeId,
      auth: {
        projectId: requestContext.projectId,
        token: requestContext.token,
        userId: requestContext.userId,
      },
      callback: (_error, response) => {
        if (requestId !== requestRef.current) return;
        pendingRequestRef.current = null;
        hideLoader();
        const knowledge = response?.getSuccess() ? response.getData() : null;
        if (knowledge) {
          loadedContextRef.current = requestContext;
          setCurrentKnowledge(knowledge);
          return;
        }
        setCurrentKnowledgeStatus(hasCurrentKnowledge ? 'ready' : 'error');
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
    return () => {
      requestRef.current += 1;
      if (pendingRequestRef.current !== null) {
        pendingRequestRef.current = null;
        hideLoader();
      }
    };
  }, [hideLoader, onReload]);

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

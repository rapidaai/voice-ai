import React, { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { useCredential } from '@/hooks/use-credential';
import { useRapidaStore } from '@/stores/app';
import { TablePagination } from '@/app/components/ui/table';
import { SearchIconInput } from '@/app/components/ui/composites';
import { BluredWrapper } from '@/app/components/layout/wrapper/blured-wrapper';
import toast from 'react-hot-toast/headless';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';
import { Knowledge } from '@rapidaai/react';
import { PageLoading } from '@/app/components/ui/feedback';
import { ClickableKnowledgeCard } from '@/app/components/domain/cards/knowledge-card';
import { EmptyState } from '@/app/components/ui/feedback';
import { HowKnowledgeWorksDialog } from '@/app/components/dialogs/knowledge';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { GhostButton, PrimaryButton } from '@/app/components/ui/primitives';
import { Add, Renew } from '@carbon/icons-react';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/layout/blocks/page-title-block';
import { cn } from '@/utils';
import { PaginationButtonBlock } from '@/app/components/layout/blocks/pagination-button-block';

/**
 * Knowledge base page
 * @returns
 */
export function KnowledgePage() {
  /**
   *
   */
  const { goToCreateKnowledge } = useGlobalNavigation();
  const [userId, token, projectId] = useCredential();
  const knowledgeActions = useKnowledgePageStore();
  const { loading, showLoader, hideLoader } = useRapidaStore();
  const [hiw, sethiw] = useState(false);

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);
  const onSuccess = useCallback((data: Knowledge[]) => {
    hideLoader();
  }, []);
  /**
   * call the api
   */
  const getKnowledges = useCallback((projectId, token, userId) => {
    showLoader();
    knowledgeActions.getAllKnowledge(
      projectId,
      token,
      userId,
      onError,
      onSuccess,
    );
  }, []);

  useEffect(() => {
    getKnowledges(projectId, token, userId);
  }, [
    projectId,
    knowledgeActions.page,
    knowledgeActions.pageSize,
    knowledgeActions.criteria,
  ]);

  return (
    <div className={cn('flex flex-col h-full flex-1 overflow-auto')}>
      <Helmet title="Knowledge" />
      <HowKnowledgeWorksDialog setModalOpen={sethiw} modalOpen={hiw} />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Knowledge bases</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${knowledgeActions.knowledgeBases.length}/${knowledgeActions.totalCount}`}
          </div>
        </div>
        {/* Carbon UI Shell toolbar pattern */}
        <div className="flex items-stretch h-12 border-l border-border-subtle ">
          <GhostButton
            size="md"
            type="button"
            onClick={() => sethiw(!hiw)}
            className="h-full whitespace-nowrap border-r border-border-subtle"
          >
            How it works?
          </GhostButton>
          <PrimaryButton
            size="md"
            type="button"
            onClick={() => goToCreateKnowledge()}
            className="h-full whitespace-nowrap"
            renderIcon={Add}
          >
            Add new knowledge
          </PrimaryButton>
        </div>
      </PageHeaderBlock>

      <BluredWrapper className="p-0">
        <SearchIconInput className="bg-light-background" />
        <PaginationButtonBlock>
          <TablePagination
            currentPage={knowledgeActions.page}
            onChangeCurrentPage={knowledgeActions.setPage}
            totalItem={knowledgeActions.totalCount}
            pageSize={knowledgeActions.pageSize}
            onChangePageSize={knowledgeActions.setPageSize}
          />
          <GhostButton
            size="md"
            hasIconOnly
            iconDescription="Refresh knowledge bases"
            renderIcon={Renew}
            onClick={() => {
              getKnowledges(projectId, token, userId);
            }}
          />
        </PaginationButtonBlock>
      </BluredWrapper>

      {loading ? (
        <PageLoading className="h-full" />
      ) : knowledgeActions.knowledgeBases &&
        knowledgeActions.knowledgeBases.length > 0 ? (
        <section className="grid content-start grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 gap-4 grow shrink-0 m-4">
          {knowledgeActions.knowledgeBases.map((kf, idx) => (
            <ClickableKnowledgeCard key={idx} knowledge={kf} />
          ))}
        </section>
      ) : knowledgeActions.criteria.length > 0 ? (
        <div className="flex flex-col h-full flex-1 items-center justify-center">
          <EmptyState
            title="No knowledge bases found"
            subtitle="No knowledge bases match your current filters."
            action="Create new knowledge"
            onAction={goToCreateKnowledge}
          />
        </div>
      ) : (
        <div className="flex flex-col h-full flex-1 items-center justify-center">
          <EmptyState
            title="No knowledge bases"
            subtitle="Create a private knowledge base for a client, brand, or internal team."
            action="Create new knowledge"
            onAction={goToCreateKnowledge}
          />
        </div>
      )}
    </div>
  );
}

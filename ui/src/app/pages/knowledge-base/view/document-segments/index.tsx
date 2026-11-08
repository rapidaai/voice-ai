import { KnowledgeDocumentSegment } from '@rapidaai/react';
import { TablePagination } from '@/app/components/ui/table';
import { BluredWrapper } from '@/app/components/layout/wrapper/blured-wrapper';
import { BaseCard } from '@/app/components/ui/primitives';
import { useRapidaStore } from '@/stores/app';
import { useKnowledgeDocumentSegmentPageStore } from '@/stores/knowledge/knowledge-document-segment.store';
import { FC, useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Knowledge } from '@rapidaai/react';
import { EditButton } from '@/app/components/ui/primitives';
import { DeleteButton } from '@/app/components/ui/primitives';
import { useCurrentCredential } from '@/hooks/use-credential';

import { EditKnowledgeDocumentSegmentDialog } from '@/app/components/dialogs/knowledge';
import { DeleteKnowledgeDocumentSegmentDialog } from '@/app/components/dialogs/knowledge';
import { EmptyState } from '@/app/components/ui/feedback';

export const DocumentSegments: FC<{
  currentKnowledge: Knowledge;
  onAddKnowledgeDocument: () => void;
}> = ck => {
  const { authId, token, projectId } = useCurrentCredential();
  const knowledgeDocumentActions = useKnowledgeDocumentSegmentPageStore();
  const { showLoader, hideLoader } = useRapidaStore();

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: KnowledgeDocumentSegment[]) => {
    hideLoader();
  }, []);

  const getKnowledgeDocumentSegments = useCallback(
    (id, projectId, token, userId) => {
      showLoader();
      knowledgeDocumentActions.getAllKnowledgeDocumentSegment(
        id,
        projectId,
        token,
        userId,
        onError,
        onSuccess,
      );
    },
    [],
  );

  useEffect(() => {
    getKnowledgeDocumentSegments(
      ck.currentKnowledge.getId(),
      projectId,
      token,
      authId,
    );
  }, [
    projectId,
    knowledgeDocumentActions.page,
    knowledgeDocumentActions.pageSize,
    knowledgeDocumentActions.criteria,
  ]);

  const [editingSegment, setEditingSegment] =
    useState<KnowledgeDocumentSegment | null>(null);
  const [deletingSegment, setDeletingSegment] =
    useState<KnowledgeDocumentSegment | null>(null);

  return (
    <>
      {knowledgeDocumentActions.knowledgeDocumentSegments &&
      knowledgeDocumentActions.knowledgeDocumentSegments.length > 0 ? (
        <>
          <BluredWrapper className="p-0">
            <TablePagination
              currentPage={knowledgeDocumentActions.page}
              onChangeCurrentPage={knowledgeDocumentActions.setPage}
              totalItem={knowledgeDocumentActions.totalCount}
              pageSize={knowledgeDocumentActions.pageSize}
              onChangePageSize={knowledgeDocumentActions.setPageSize}
            />
          </BluredWrapper>

          <div className="grid content-start grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4 grow shrink-0 m-4">
            {knowledgeDocumentActions.knowledgeDocumentSegments.map(
              (segment, index) => (
                <BaseCard key={index} className="p-6 gap-6">
                  {/* Segment header */}
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono text-muted ">
                      {segment.getDocumentId().substring(0, 12)}
                    </span>
                    <div className="flex">
                      <EditButton onClick={() => setEditingSegment(segment)} />
                      <DeleteButton
                        onClick={() => setDeletingSegment(segment)}
                      />
                    </div>
                  </div>

                  {/* Segment text */}
                  <div className="text-sm text-foreground leading-[20px] flex-1">
                    {parseMarkdown(segment.getText())}
                  </div>

                  {/* Entity tags */}
                  {Object.entries(segment.getEntities()?.toObject() || {}).some(
                    ([, values]) => Array.isArray(values) && values.length > 0,
                  ) && (
                    <div className="flex flex-col gap-3 text-sm border-t border-border-subtle pt-4">
                      {Object.entries(
                        segment.getEntities()?.toObject() || {},
                      ).map(
                        ([key, values]) =>
                          Array.isArray(values) &&
                          values.length > 0 && (
                            <div key={key} className="flex flex-col gap-2">
                              <span className="text-[10px] font-semibold tracking-[0.12em] uppercase text-muted ">
                                {key
                                  .replace('List', '')
                                  .replace(/([A-Z])/g, ' $1')
                                  .trim()}
                              </span>
                              <div className="flex flex-wrap gap-1">
                                {values.map((value, i) => (
                                  <span
                                    key={i}
                                    className="inline-flex items-center h-6 px-3 text-xs font-medium bg-layer text-muted "
                                  >
                                    {value}
                                  </span>
                                ))}
                              </div>
                            </div>
                          ),
                      )}
                    </div>
                  )}
                </BaseCard>
              ),
            )}
          </div>

          {editingSegment && (
            <EditKnowledgeDocumentSegmentDialog
              segment={editingSegment}
              onClose={() => setEditingSegment(null)}
              onUpdate={() => {
                getKnowledgeDocumentSegments(
                  ck.currentKnowledge.getId(),
                  projectId,
                  token,
                  authId,
                );
              }}
            />
          )}

          {deletingSegment && (
            <DeleteKnowledgeDocumentSegmentDialog
              segment={deletingSegment}
              onClose={() => setDeletingSegment(null)}
              onDelete={() => {
                getKnowledgeDocumentSegments(
                  ck.currentKnowledge.getId(),
                  projectId,
                  token,
                  authId,
                );
              }}
            />
          )}
        </>
      ) : (
        <div className="flex flex-col h-full flex-1 items-center justify-center">
          <EmptyState
            title="No segments"
            subtitle="There are no document segments in this knowledge base."
            action="Add new document"
            onAction={() => ck.onAddKnowledgeDocument()}
          />
        </div>
      )}
    </>
  );
};

function parseMarkdown(text: string): React.ReactNode {
  // Bold
  text = text.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
  // Italic
  text = text.replace(/\*(.*?)\*/g, '<em>$1</em>');
  // Links
  text = text.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');

  return <span dangerouslySetInnerHTML={{ __html: text }} />;
}

import { KnowledgeDocumentSegment } from '@rapidaai/react';
import { PageHeading } from '@/app/components/Heading/PageHeading';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { useRapidaStore } from '@/hooks';
import { useKnowledgeDocumentSegmentPageStore } from '@/hooks/use-knowledge-document-segment-page-store';
import { cn } from '@/styles/media';
import { FC, useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Knowledge } from '@rapidaai/react';
import { EditButton } from '@/app/components/Form/Button/edit-button';
import { DeleteButton } from '@/app/components/Form/Button/delete-button';
import { useCurrentCredential } from '@/hooks/use-credential';

import { EditKnowledgeDocumentSegmentDialog } from '@/app/components/base/modal/edit-knowledge-document-segment-modal';
import { DeleteKnowledgeDocumentSegmentDialog } from '@/app/components/base/modal/delete-knowledge-document-segment-modal';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
export const DocumentSegments: FC<{
  currentKnowledge: Knowledge;
}> = ck => {
  const { authId, token, projectId } = useCurrentCredential();

  const navigator = useGlobalNavigation();
  const knowledgeDocumentActions = useKnowledgeDocumentSegmentPageStore();
  const { showLoader, hideLoader } = useRapidaStore();
  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);
  const onSuccess = useCallback((data: KnowledgeDocumentSegment[]) => {
    console.dir(data);
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
            <PageHeading className="flex items-center space-x-2"></PageHeading>
            <TablePagination
              currentPage={knowledgeDocumentActions.page}
              onChangeCurrentPage={knowledgeDocumentActions.setPage}
              totalItem={knowledgeDocumentActions.totalCount}
              pageSize={knowledgeDocumentActions.pageSize}
              onChangePageSize={knowledgeDocumentActions.setPageSize}
            />
          </BluredWrapper>
          <div className="grid content-start grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 grow shrink-0 px-4 py-4">
            {knowledgeDocumentActions.knowledgeDocumentSegments.map(
              (segment, index) => {
                return (
                  <div
                    key={index}
                    className={cn(
                      'flex flex-col h-full p-6',
                      'space-y-6',
                      'shrink-0',
                      'shadow-sm hover:shadow-lg',
                      'bg-white dark:bg-gray-950/20 rounded-[2px] border border-gray-200 dark:border-gray-800 col-span-1',
                    )}
                  >
                    <div className="flex justify-between">
                      <div>ID: {segment.getDocumentId().substring(0, 12)}</div>
                      <div className="flex">
                        <EditButton
                          onClick={() => setEditingSegment(segment)}
                        />
                        <DeleteButton
                          onClick={() => setDeletingSegment(segment)}
                        />
                      </div>
                    </div>
                    {/* line-clamp-5 */}
                    <div className={cn('text-base')}>
                      {parseMarkdown(segment.getText())}
                    </div>
                    <div className="space-y-3 text-sm">
                      {Object.entries(
                        segment.getEntities()?.toObject() || {},
                      ).map(
                        ([key, values]) =>
                          Array.isArray(values) &&
                          values.length > 0 && (
                            <div key={key} className="space-y-3">
                              <div className="font-semibold uppercase tracking-wider">
                                {key
                                  .replace('List', '')
                                  .replace(/([A-Z])/g, ' $1')
                                  .trim()}
                              </div>
                              <div className="flex flex-wrap gap-1">
                                {values.map((value, index) => (
                                  <span
                                    key={index}
                                    className="px-4 py-1.5  border"
                                  >
                                    {value}
                                  </span>
                                ))}
                              </div>
                            </div>
                          ),
                      )}
                    </div>
                  </div>
                );
              },
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
        <div className="flex flex-col items-center justify-center">
          <ActionableEmptyMessage
            title="No Documents"
            subtitle="There are no document segments in knowledge to display"
            action="Add New Document"
            onActionClick={() =>
              navigator.goToKnowledgeAddManualFile(ck.currentKnowledge.getId())
            }
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

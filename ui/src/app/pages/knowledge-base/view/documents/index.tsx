import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import React, { useCallback, useEffect } from 'react';
import toast from 'react-hot-toast/headless';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { KnowledgeDocument } from '@rapidaai/react';
import { useKnowledgeDocumentPageStore } from '@/hooks/use-knowledge-document-page-store';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { SingleDocument } from '@/app/pages/knowledge-base/view/documents/single-document';
import { useNavigate } from 'react-router-dom';
import { Knowledge } from '@rapidaai/react';
import { Spinner } from '@/app/components/Loader/Spinner';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { ScrollableResizableTable } from '@/app/components/data-table';

export function Documents(props: { currentKnowledge: Knowledge }) {
  const [userId, token, projectId] = useCredential();
  const navigator = useNavigate();
  const rapidaContext = useRapidaStore();
  const knowledgeDocumentAction = useKnowledgeDocumentPageStore();

  const getKnowledgeDocument = useCallback(() => {
    knowledgeDocumentAction.getAllKnowledgeDocument(
      props.currentKnowledge.getId(),
      projectId,
      token,
      userId,
      (err: string) => {
        rapidaContext.hideLoader();
        toast.error(err);
      },
      (data: KnowledgeDocument[]) => {
        rapidaContext.hideLoader();
      },
    );
  }, [props.currentKnowledge]);

  useEffect(() => {
    rapidaContext.showLoader();
    getKnowledgeDocument();
  }, [
    props.currentKnowledge,
    projectId,
    knowledgeDocumentAction.page,
    knowledgeDocumentAction.pageSize,
    knowledgeDocumentAction.criteria,
  ]);

  return (
    <>
      {knowledgeDocumentAction.documents &&
      knowledgeDocumentAction.documents.length > 0 ? (
        <div className="flex flex-col h-full flex-1">
          <BluredWrapper className="p-0">
            <SearchIconInput className="bg-light-background" />
            <TablePagination
              columns={knowledgeDocumentAction.columns}
              currentPage={knowledgeDocumentAction.page}
              onChangeCurrentPage={knowledgeDocumentAction.setPage}
              totalItem={knowledgeDocumentAction.totalCount}
              pageSize={knowledgeDocumentAction.pageSize}
              onChangePageSize={knowledgeDocumentAction.setPageSize}
              onChangeColumns={knowledgeDocumentAction.setColumns}
            />
          </BluredWrapper>

          <ScrollableResizableTable
            isActionable={false}
            isOptionable={true}
            clms={knowledgeDocumentAction.columns.filter(x => {
              return x.visible;
            })}
          >
            {knowledgeDocumentAction.documents.map((kd, idx) => {
              return (
                <SingleDocument
                  key={`document_row_${idx}`}
                  document={kd}
                  onReload={() => {
                    getKnowledgeDocument();
                  }}
                />
              );
            })}
            {/* </TBody> */}
          </ScrollableResizableTable>
        </div>
      ) : knowledgeDocumentAction.documents.length > 0 ? (
        <div className="flex flex-col h-full flex-1 items-center justify-center">
          <ActionableEmptyMessage
            title="No documents"
            subtitle=" There are no documents matching with your criteria."
            action="Add New Document"
            onActionClick={() =>
              navigator(
                `/knowledge/${props.currentKnowledge.getId()}/add-knowledge-file`,
              )
            }
          />
        </div>
      ) : !rapidaContext.loading ? (
        <div className="flex flex-col h-full flex-1 items-center justify-center">
          <ActionableEmptyMessage
            title="No Documents"
            subtitle="There are no documents in knowledge to display"
            action="Add New Document"
            onActionClick={() =>
              navigator(
                `/knowledge/${props.currentKnowledge.getId()}/add-knowledge-file`,
              )
            }
          />
        </div>
      ) : (
        <div className="h-full flex justify-center items-center grow">
          <Spinner size="md" />
        </div>
      )}
    </>
  );
}

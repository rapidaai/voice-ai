import { create } from 'zustand';
import { initialPaginated } from '@/types/types.paginated';
import {
  KnowledgeDocumentProperty,
  KnowledgeDocumentType,
} from '@/types/types.knowledge-document';
import {
  ConnectionConfig,
  GetAllKnowledgeDocumentResponse,
  KnowledgeDocument,
} from '@rapidaai/react';
import { GetAllKnowledgeDocument } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { IndexKnowledgeDocument } from '@rapidaai/react';
import { IndexKnowledgeDocumentResponse } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const intialKnowledgeDocumentProperty: KnowledgeDocumentProperty = {
  /**
   * list of assistant
   */
  documents: [],
};

/**
 *
 */
export const useKnowledgeDocumentPageStore = create<KnowledgeDocumentType>(
  (set, get) => ({
    ...intialKnowledgeDocumentProperty,
    ...initialPaginated,

    /**
     * columns
     */
    columns: [
      { name: 'Document Status', key: 'getStatus', visible: true },
      { name: 'Name', key: 'getName', visible: true },
      { name: 'Document Type', key: 'getDocumenttype', visible: false },
      { name: 'Document Source', key: 'getDocumentSource', visible: true },
      { name: 'Document Size', key: 'getDocumentsize', visible: true },
      { name: 'Retrieval Count', key: 'getRetrievalcount', visible: true },
      { name: 'Token Count', key: 'getTokencount', visible: true },
      { name: 'Word Count', key: 'getWordcount', visible: true },
      { name: 'Id', key: 'getId', visible: true },
    ],
    /**
     *
     * @param number
     * @returns
     */
    setPageSize: (pageSize: number) => {
      // when someone change pagesize change the page to zero
      set({
        page: 1,
        pageSize: pageSize,
      });
    },

    /**
     *
     * @param number
     * @returns
     */
    setPage: (pg: number) => {
      set({
        page: pg,
      });
    },

    /**
     *
     * @param number
     * @returns
     */
    setTotalCount: (tc: number) => {
      set({
        totalCount: tc,
      });
    },

    /**
     *
     * @param k
     * @param v
     */
    addCriteria: (k: string, v: string, logic: string) => {
      get().criteria.push({ key: k, value: v, logic: logic });
    },

    setKnowledgeDocuments: (dc: KnowledgeDocument[]) => {
      set({
        documents: dc,
      });
    },
    /**
     *
     * @param projectId
     * @param token
     * @param userId
     */
    getAllKnowledgeDocument: (
      knowledgeId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: KnowledgeDocument[]) => void,
    ) => {
      const afterGetAllKnowledgeDocument = (
        err: ServiceError | null,
        gur: GetAllKnowledgeDocumentResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          get().setKnowledgeDocuments(gur.getDataList());
          let paginated = gur.getPaginated();
          if (paginated) {
            get().setTotalCount(paginated.getTotalitem());
          }
          onSuccess(gur.getDataList());
        } else {
          let errorMessage = gur?.getError();
          if (errorMessage) {
            onError(errorMessage.getHumanmessage());
            return;
          }
          onError(
            'Unable to get all the documents for knowledgebase, please try again later.',
          );
        }
      };

      GetAllKnowledgeDocument(
        connectionConfig,
        knowledgeId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllKnowledgeDocument,
        ConnectionConfig.WithDebugger({
          authorization: token,
          projectId: projectId,
          userId: userId,
        }),
      );
    },

    indexKnowledgeDocument: (
      knowledgeId: string,
      knowledgeDocumentId: string[],
      indexType: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: boolean) => void,
    ) => {
      const afterIndexKnowledgeDocument = (
        err: ServiceError | null,
        gur: IndexKnowledgeDocumentResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          onSuccess(gur.getSuccess());
        } else {
          onError(
            'Unable to build the index for the document, please try again later.',
          );
        }
      };

      IndexKnowledgeDocument(
        connectionConfig,
        knowledgeId,
        knowledgeDocumentId,
        indexType,
        ConnectionConfig.WithDebugger({
          authorization: token,
          projectId: projectId,
          userId: userId,
        }),
        afterIndexKnowledgeDocument,
      );
    },

    /**
     *
     * @param cl
     */
    setColumns(cl: { name: string; key: string; visible: boolean }[]) {
      set({
        columns: cl,
      });
    },

    /**
     *
     * @param k
     * @returns
     */
    visibleColumn: (k: string): boolean => {
      const column = get().columns.find(c => c.key === k);
      return column ? column.visible : false;
    },

    /**
     * clear everything from the context
     * @returns
     */
    clear: () => set({ ...intialKnowledgeDocumentProperty }, false),
  }),
);

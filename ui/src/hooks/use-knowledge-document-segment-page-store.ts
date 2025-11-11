import { create } from 'zustand';
import {
  KnowledgeDocumentSegmentType,
  KnowledgeDocumentSegmentTypeProperty,
} from '@/types';
import { initialPaginated } from '@/types/types.paginated';
import {
  GetAllKnowledgeDocumentSegmentResponse,
  KnowledgeDocument,
  KnowledgeDocumentSegment,
} from '@rapidaai/react';
import { GetAllKnowledgeDocumentSegment } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const initialKnowledgeDocumentSegmentType: KnowledgeDocumentSegmentTypeProperty =
  {
    currentKnowledgeDocument: null,
    /**
     * list of assistant
     */
    knowledgeDocumentSegments: [],
  };

/**
 *
 */
export const useKnowledgeDocumentSegmentPageStore =
  create<KnowledgeDocumentSegmentType>((set, get) => ({
    ...initialKnowledgeDocumentSegmentType,
    ...initialPaginated,

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
     * @param ep
     */
    setKnowledgeDocumentSegments: (ep: KnowledgeDocumentSegment[]) => {
      set({
        knowledgeDocumentSegments: ep,
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

    onChangeCurrentKnowledgeDocument: (kd: KnowledgeDocument) => {
      set({
        currentKnowledgeDocument: kd,
      });
    },

    /**
     *
     * @param projectId
     * @param token
     * @param userId
     */
    getAllKnowledgeDocumentSegment: (
      knowledgeId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: KnowledgeDocumentSegment[]) => void,
    ) => {
      // let id = get().currentKnowledgeDocument?.getId();
      const afterGetAllKnowledgeDocumentSegment = (
        err: ServiceError | null,
        gur: GetAllKnowledgeDocumentSegmentResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          get().setKnowledgeDocumentSegments(gur.getDataList());
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
            'Unable to get your knowledge segments, please try again later.',
          );
        }
      };

      GetAllKnowledgeDocumentSegment(
        connectionConfig,
        knowledgeId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllKnowledgeDocumentSegment,
        {
          authorization: token,
          'x-project-id': projectId,
          'x-auth-id': userId,
        },
      );
    },

    /**
     * columns
     */
    columns: [],

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
    clear: () => set({}, true),
  }));

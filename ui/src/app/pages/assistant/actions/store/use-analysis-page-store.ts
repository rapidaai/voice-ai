import { create } from 'zustand';
import { initialPaginated } from '@/types/types.paginated';
import { ServiceError } from '@rapidaai/react';
import {
  AssistantAnalysis,
  GetAllAssistantAnalysisResponse,
  GetAssistantAnalysisResponse,
} from '@rapidaai/react';
import {
  AssistantAnalysisProperty,
  AssistantAnalysisType,
} from './types/types.assistant-analysis';
import {
  DeleteAssistantAnalysis,
  GetAllAssistantAnalysis,
} from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const initialAssistantAnalysis: AssistantAnalysisProperty = {
  analysises: [],
};

/**
 *
 */
export const useAssistantAnalysisPageStore = create<AssistantAnalysisType>(
  (set, get) => ({
    ...initialAssistantAnalysis,
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
    onChangeAssistantAnalysises: (ep: AssistantAnalysis[]) => {
      set({
        analysises: ep,
      });
    },

    /**
     *
     * @param k
     * @param v
     */
    addCriteria: (k: string, v: string, logic: string) => {
      let current = get().criteria.filter(
        x => x.key !== k && x.logic !== logic,
      );
      if (v) current.push({ key: k, value: v, logic: logic });
      set({
        criteria: current,
      });
    },

    /**
     *
     * @param v
     */
    addCriterias: (v: { k: string; v: string; logic: string }[]) => {
      let current = get().criteria.filter(
        x => !v.find(y => y.k === x.key && x.logic === y.logic),
      );
      v.forEach(c => {
        current.push({ key: c.k, value: c.v, logic: c.logic });
      });
      set({
        criteria: current,
      });
    },

    /**
     *
     * @param projectId
     * @param token
     * @param userId
     */
    getAssistantAnalysis: (
      assistantId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: AssistantAnalysis[]) => void,
    ) => {
      const afterGetAllAssistantAnalysis = (
        err: ServiceError | null,
        gur: GetAllAssistantAnalysisResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          get().onChangeAssistantAnalysises(gur.getDataList());
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
          onError('Unable to get your activity log, please try again later.');
        }
      };

      GetAllAssistantAnalysis(
        connectionConfig,
        assistantId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllAssistantAnalysis,
        {
          authorization: token,
          'x-project-id': projectId,
          'x-auth-id': userId,
        },
      );
    },

    /**
     *
     * @param assistantId
     * @param analysisId
     * @param projectId
     * @param token
     * @param userId
     * @param onError
     * @param onSuccess
     */
    deleteAssistantAnalysis: (
      assistantId: string,
      analysisId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: AssistantAnalysis) => void,
    ) => {
      const afterDeleteAssistantAnalysis = (
        err: ServiceError | null,
        gur: GetAssistantAnalysisResponse | null,
      ) => {
        if (gur?.getSuccess() && gur.getData()) {
          onSuccess(gur.getData()!);
        } else {
          let errorMessage = gur?.getError();
          if (errorMessage) {
            onError(errorMessage.getHumanmessage());
            return;
          }
          onError(
            'Unable to delete assistant analysis, please try again later.',
          );
        }
      };

      DeleteAssistantAnalysis(
        connectionConfig,
        assistantId,
        analysisId,
        afterDeleteAssistantAnalysis,
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
    columns: [
      { name: 'ID', key: 'id', visible: false },
      { name: 'Name', key: 'name', visible: true },
      { name: 'EndpointID', key: 'endpointId', visible: true },
      { name: 'EndpointVersion', key: 'endpointVersion', visible: false },
      { name: 'ExecutionPriority', key: 'executionPriority', visible: true },
      { name: 'Status', key: 'status', visible: true },
      { name: 'Created Date', key: 'created_date', visible: true },
    ],

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
    clear: () => set({ ...initialAssistantAnalysis, ...initialPaginated }),
  }),
);

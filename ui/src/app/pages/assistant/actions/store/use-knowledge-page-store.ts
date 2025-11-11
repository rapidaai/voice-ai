import { create } from 'zustand';
import { initialPaginated } from '@/types/types.paginated';
import { ServiceError } from '@rapidaai/react';
import {
  AssistantKnowledge,
  GetAllAssistantKnowledgeResponse,
  GetAssistantKnowledgeResponse,
} from '@rapidaai/react';
import {
  AssistantKnowledgeProperty,
  AssistantKnowledgeType,
} from './types/types.assistant-knowledge';
import {
  DeleteAssistantKnowledge,
  GetAllAssistantKnowledge,
} from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const initialAssistantKnowledge: AssistantKnowledgeProperty = {
  knowledges: [],
};

/**
 *
 */
export const useAssistantKnowledgePageStore = create<AssistantKnowledgeType>(
  (set, get) => ({
    ...initialAssistantKnowledge,
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
    onChangeAssistantKnowledges: (ep: AssistantKnowledge[]) => {
      set({
        knowledges: ep,
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
    getAssistantKnowledge: (
      assistantId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: AssistantKnowledge[]) => void,
    ) => {
      const afterGetAllAssistantKnowledge = (
        err: ServiceError | null,
        gur: GetAllAssistantKnowledgeResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          get().onChangeAssistantKnowledges(gur.getDataList());
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

      GetAllAssistantKnowledge(
        connectionConfig,
        assistantId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllAssistantKnowledge,
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
     * @param knowledgeId
     * @param projectId
     * @param token
     * @param userId
     * @param onError
     * @param onSuccess
     */
    deleteAssistantKnowledge: (
      assistantId: string,
      knowledgeId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: AssistantKnowledge) => void,
    ) => {
      const afterDeleteAssistantKnowledge = (
        err: ServiceError | null,
        gur: GetAssistantKnowledgeResponse | null,
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
            'Unable to delete assistant knowledge, please try again later.',
          );
        }
      };

      DeleteAssistantKnowledge(
        connectionConfig,
        assistantId,
        knowledgeId,
        afterDeleteAssistantKnowledge,
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
      { name: 'HTTP Endpoint', key: 'httpUrl', visible: true },
      { name: 'Max Retry Count', key: 'maxRetryCount', visible: false },
      { name: 'Timeout Seconds', key: 'timeoutSeconds', visible: false },
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
    clear: () => set({ ...initialAssistantKnowledge, ...initialPaginated }),
  }),
);

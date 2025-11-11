import {
  ConversationLog,
  ConversationLogProperty,
} from '@/types/types.conversation-log';
import {
  initialPaginated,
  initialPaginatedState,
} from '@/types/types.paginated';
import { create } from 'zustand';
import { ServiceError } from '@rapidaai/react';
import { GetAllMessageResponse } from '@rapidaai/react';
import { GetMessages } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
const initialState: ConversationLogProperty = {
  /**
   *
   */
  assistantMessages: [],

  /**
   *
   */
  fields: ['metadata', 'metric', 'stage', 'request', 'response'],
};

/**
 *
 */
export const useConversationLogPageStore = create<ConversationLog>(
  (set, get) => ({
    ...initialState,
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

    setFields: (
      fl: ('metadata' | 'metric' | 'stage' | 'request' | 'response')[],
    ) => {
      set({
        fields: fl,
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
     * @param assistantId
     * @param projectId
     * @param token
     * @param userId
     * @param onError
     * @param onSuccess
     */
    getMessages(projectId, token, userId, onError, onSuccess) {
      const afterGetAssistantMessages = (
        err: ServiceError | null,
        gur: GetAllMessageResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          set({
            assistantMessages: gur.getDataList(),
          });
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
            'Unable to get all assistant messages, please try again in sometime.',
          );
        }
      };

      GetMessages(
        connectionConfig,
        get().page,
        get().pageSize,
        get().criteria,
        get().fields,
        afterGetAssistantMessages,
        {
          authorization: token,
          'x-auth-id': userId,
          'x-project-id': projectId,
        },
      );
    },
    /**
     * columns
     */
    columns: [
      { name: 'MessageID', key: 'id', visible: true },
      { name: 'Version', key: 'version', visible: false },
      {
        name: 'Session ID',
        key: 'assistant_conversation_id',
        visible: true,
      },
      {
        name: 'Assistant ID',
        key: 'assistant_id',
        visible: true,
      },
      { name: 'Source', key: 'source', visible: true },
      { name: 'Request', key: 'request', visible: true },
      { name: 'Response', key: 'response', visible: true },
      { name: 'Created Date', key: 'created_date', visible: true },

      { name: 'Status', key: 'status', visible: true },
      { name: 'Time Taken', key: 'time_taken', visible: true },
      { name: 'Total Token', key: 'total_token', visible: false },
      { name: 'Feedback', key: 'user_feedback', visible: false },
      { name: 'User Feedback', key: 'user_text_feedback', visible: false },
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
    clear: () =>
      set(state => ({
        ...initialState,
        ...initialPaginatedState,
      })),

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
    setCriterias: (v: { k: string; v: string; logic: string }[]) => {
      set({
        criteria: v.map(c => {
          return { key: c.k, value: c.v, logic: c.logic };
        }),
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
  }),
);

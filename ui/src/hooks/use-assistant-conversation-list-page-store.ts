import {
  AssistantConversationList,
  AssistantConversationProperty,
} from '@/types/types.assistant-conversation-list';
import {
  initialPaginated,
  initialPaginatedState,
} from '@/types/types.paginated';
import { create } from 'zustand';
import { ServiceError } from '@rapidaai/react';
import { AssistantConversation } from '@rapidaai/react';
import { GetAllAssistantConversation } from '@rapidaai/react';
import { GetAllAssistantConversationResponse } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const initialState: AssistantConversationProperty = {
  assistantConversations: [],
  currentConversation: null,

  dialogVisible: false,
};

/**
 *
 */
export const useAssistantConversationListPageStore =
  create<AssistantConversationList>((set, get) => ({
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
     * @param conv
     */
    showDialog: (conv: AssistantConversation) => {
      set({
        currentConversation: conv,
        dialogVisible: true,
      });
    },

    hideDialog: () => {
      set({
        currentConversation: null,
        dialogVisible: false,
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

    onChangeAssistantConversations: (msg: AssistantConversation[]) => {
      set({ assistantConversations: msg });
    },

    onChangeCurrentConversation: (conv: AssistantConversation | null) => {
      set({
        currentConversation: conv,
      });
    },

    getAssistantConversations(
      assistantId,
      projectId,
      token,
      userId,
      onError,
      onSuccess,
    ) {
      const afterGetAllAssistantConversation = (
        err: ServiceError | null,
        gur: GetAllAssistantConversationResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          set({
            assistantConversations: gur.getDataList(),
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
            'Unable to get all conversation for assistant, please try again in sometime.',
          );
        }
      };

      GetAllAssistantConversation(
        connectionConfig,
        assistantId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllAssistantConversation,
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
      { name: 'Session ID', key: 'id', visible: true },
      {
        name: 'Assistant ID',
        key: 'assistant_id',
        visible: true,
      },
      {
        name: 'Assistant Version',
        key: 'assistant_provider_model_id',
        visible: false,
      },
      { name: 'Direction', key: 'direction', visible: true },
      { name: 'User Identifier', key: 'identifier', visible: true },
      { name: 'Source', key: 'source', visible: false },
      { name: 'Duration (in min)', key: 'duration', visible: true },
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
     * clear everything from the context
     * @returns
     */
    clear: () =>
      set(state => ({
        ...initialState,
        ...initialPaginatedState,
      })),
  }));

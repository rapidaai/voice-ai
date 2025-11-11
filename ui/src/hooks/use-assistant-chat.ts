import {
  ConnectionConfig,
  GetAllAssistantConversationMessage,
} from '@rapidaai/react';
import {
  AssistantConversationMessage,
  GetAllConversationMessageResponse,
} from '@rapidaai/react';

import {
  AssistantChatProperty,
  AssistantChatType,
} from '@/types/types.assistant-chat';
import {
  initialPaginated,
  initialPaginatedState,
} from '@/types/types.paginated';
import React from 'react';
import { create } from 'zustand';
import { ServiceError } from '@rapidaai/react';
import { Assistant } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const initialState: AssistantChatProperty = {
  /**
   * current assistant which will be targeted
   */
  currentAssistant: null,

  /**
   *
   */
  currentAssistantConversationId: null,

  /**
   *
   */
  conversations: [],
};

const initialChatActionState = {
  /**
   *
   * @param assistant
   * @returns
   */
  onChangeCurrentAssistant: (assistant: Assistant) => {},

  /**
   *
   * @param message
   * @returns
   */
  onChangeConversationMessages: (
    message: Array<AssistantConversationMessage>,
  ) => {},

  /**
   *
   * @param msg
   */
  onChangeConversactionMessage: (msg: AssistantConversationMessage) => {},

  /**
   *
   * @param assistantConversationId
   * @returns
   */
  onChangeAssistantConversationId: (assistantConversationId: string) => {},
};

const initialChatApiCallState = {
  onGetConversationMessages: function (
    assistantId: string,
    conversationId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConversationMessage[]) => void,
  ): void {
    throw new Error('Function not implemented.');
  },
  clear: function (): void {
    throw new Error('Function not implemented.');
  },
};

export const AssistantChatContext = React.createContext<AssistantChatType>({
  ...initialState,
  ...initialPaginated,
  ...initialChatActionState,
  ...initialChatApiCallState,
});

/**
 *
 */
export const useAssistantChat = create<AssistantChatType>((set, get) => ({
  ...initialState,
  ...initialPaginated,
  ...initialChatActionState,
  ...initialChatApiCallState,

  pageSize: 100,

  onChangeAssistantConversationId: (assistantConversationId: string) => {
    set({
      currentAssistantConversationId: assistantConversationId,
    });
  },

  /**
   *
   * @param message
   */
  onChangeConversationMessages: (
    message: Array<AssistantConversationMessage>,
  ) => {
    set({
      conversations: message,
    });
  },

  onChangeConversactionMessage: (msg: AssistantConversationMessage) => {
    set({
      conversations: [
        ...get().conversations.filter(
          x => x.getMessageid() !== msg.getMessageid(),
        ),
        msg,
      ],
    });
  },

  /**
   *
   * @param assistantId
   * @param conversationId
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onGetConversationMessages: (
    assistantId: string,
    conversationId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AssistantConversationMessage[]) => void,
  ) => {
    const afterGetAllAssistantConversationMessage = (
      err: ServiceError | null,
      epmr: GetAllConversationMessageResponse | null,
    ) => {
      if (epmr?.getSuccess()) {
        let message = epmr.getDataList();
        if (message) {
          onSuccess(message);
        }
      } else {
        const errorMessage =
          'Unable to get your assistant. please try again later.';
        const error = epmr?.getError();
        if (error) {
          onError(error.getHumanmessage());
          return;
        }
        onError(errorMessage);
        return;
      }
    };

    GetAllAssistantConversationMessage(
      connectionConfig,
      assistantId,
      conversationId,
      get().page,
      get().pageSize,
      get().criteria,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
      afterGetAllAssistantConversationMessage,
    );
  },

  /**
   *
   * @returns
   */
  onGetMessages: () => {},
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

  onChangeCurrentAssistant: (assistant: Assistant) => {
    set({ currentAssistant: assistant, conversations: [] });
  },

  /**
   *
   * @param k
   * @param v
   */
  addCriteria: (k: string, v: string, logic: string) => {
    let current = get().criteria.filter(x => x.key !== k && x.logic !== logic);
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
   * columns
   */
  columns: [],

  visibleColumn: str => {
    return true;
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
   * clear everything from the context
   * @returns
   */
  clear: () => set({ ...initialState, ...initialPaginatedState }, false),
}));

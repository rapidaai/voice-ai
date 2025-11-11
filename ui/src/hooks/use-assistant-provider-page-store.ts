import {
  AssistantDefinition,
  ConnectionConfig,
  GetAllAssistantProvider,
  GetAssistant,
  GetAssistantRequest,
  UpdateAssistantVersion,
  UpdateAssistantVersionRequest,
} from '@rapidaai/react';
import {
  Assistant,
  GetAllAssistantProviderResponse,
  GetAssistantResponse,
} from '@rapidaai/react';

import { AssistantProviderType, AssistantProviderTypeProperty } from '@/types';
import { ServiceError } from '@rapidaai/react';
import {
  initialPaginated,
  initialPaginatedState,
} from '@/types/types.paginated';
import { create } from 'zustand';
import { connectionConfig } from '@/configs';

const initialState: AssistantProviderTypeProperty = {
  /**
   * current assistant which will be targeted
   */
  assistant: null,

  /**
   * list of assistant where these will be part of it
   */
  assistantProviders: [],
};

/**
 *
 */
export const useAssistantProviderPageStore = create<AssistantProviderType>(
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
     * @param assistant
     */
    onChangeAssistant: (assistant: Assistant) => {
      set({ assistant: assistant });
    },

    /**
     *
     * @param assistantProviderModels
     */
    setAssistantProviderModels: (
      assistantProviderModels: GetAllAssistantProviderResponse.AssistantProvider[],
    ) => {
      set({ assistantProviders: assistantProviderModels });
    },

    /**
     *
     * @param k
     * @param v
     */
    addCriteria: (k: string, v: string, logic: string) => {
      get().criteria.push({ key: k, value: v, logic: logic });
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

    getAssistantProviders: (
      assistantId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (
        e: GetAllAssistantProviderResponse.AssistantProvider[],
      ) => void,
    ) => {
      const afterGetAllAssistantProviderModel = (
        err: ServiceError | null,
        gur: GetAllAssistantProviderResponse | null,
      ) => {
        if (gur?.getSuccess()) {
          set({
            assistantProviders: gur.getDataList(),
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
          onError('Unable to get all the version of assistant.');
        }
      };

      GetAllAssistantProvider(
        connectionConfig,
        assistantId,
        get().page,
        get().pageSize,
        get().criteria,
        afterGetAllAssistantProviderModel,
        {
          authorization: token,
          'x-auth-id': userId,
          'x-project-id': projectId,
        },
      );
    },

    /**
     *
     * @param projectId
     * @param token
     * @param userId
     */
    getAssistant: (
      assistantId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: Assistant) => void,
    ) => {
      const request = new GetAssistantRequest();
      const assistantDef = new AssistantDefinition();
      assistantDef.setAssistantid(assistantId);
      request.setAssistantdefinition(assistantDef);
      GetAssistant(
        connectionConfig,
        request,
        ConnectionConfig.WithDebugger({
          authorization: token,
          userId: userId,
          projectId: projectId,
        }),
      )
        .then(gur => {
          if (gur?.getSuccess()) {
            let ast = gur.getData();
            if (ast) {
              get().onChangeAssistant(ast);
              onSuccess(ast);
              return;
            }
          } else {
            let errorMessage = gur?.getError();
            if (errorMessage) {
              onError(errorMessage.getHumanmessage());
              return;
            }
            onError('Unable to get your assistant, please try again later.');
          }
        })
        .catch(err => {
          onError('Unable to get your assistant, please try again later.');
        });
    },

    onReleaseVersion: (
      assistantProvider: string,
      assistantProviderId: string,
      projectId: string,
      token: string,
      userId: string,
      onError: (err: string) => void,
      onSuccess: (e: Assistant) => void,
    ) => {
      let assistant = get().assistant;
      if (!assistant && assistant === null) {
        return;
      }

      if (assistant?.getAssistantproviderid() === assistantProviderId) {
        onSuccess(assistant);
        return;
      }

      const rqs = new UpdateAssistantVersionRequest();
      rqs.setAssistantid(assistant?.getId());
      rqs.setAssistantprovider(assistantProvider);
      rqs.setAssistantproviderid(assistantProviderId);
      UpdateAssistantVersion(
        connectionConfig,
        rqs,
        ConnectionConfig.WithDebugger({
          authorization: token,
          userId: userId,
          projectId: projectId,
        }),
      )
        .then((aur: GetAssistantResponse) => {
          if (aur?.getSuccess()) {
            const ed = aur.getData();
            if (ed) onSuccess(ed);
          } else {
            let _er = aur?.getError();
            if (_er) {
              onError(_er.getHumanmessage());
              return;
            }
            onError('Unable to process your request. please try again later.');
            return;
          }
        })
        .catch(err => {
          onError('Unable to process your request. please try again later.');
        });
    },

    /**
     * columns
     */
    columns: [
      { name: 'Version', key: 'version', visible: true },
      { name: 'Provider', key: 'provider', visible: true },
      { name: 'Change description', key: 'change_description', visible: true },
      { name: 'Created by', key: 'created_by', visible: true },
      { name: 'Created on', key: 'created_on', visible: true },
      { name: 'Action', key: 'action', visible: true },
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
      set({
        ...initialState,
        ...initialPaginatedState,
      }),
  }),
);

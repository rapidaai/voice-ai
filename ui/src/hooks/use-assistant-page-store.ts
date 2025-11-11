import {
  AssistantDefinition,
  ConnectionConfig,
  CreateAssistantTag,
  GetAllAssistant,
  GetAssistant,
  GetAssistantRequest,
  UpdateAssistantDetail,
} from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import {
  Assistant,
  GetAllAssistantResponse,
  GetAssistantResponse,
} from '@rapidaai/react';

import { create } from 'zustand';
import { AssistantType, AssistantTypeProperty } from '@/types';
import {
  initialPaginated,
  initialPaginatedState,
} from '@/types/types.paginated';
import { connectionConfig } from '@/configs';

const intialAssistant: AssistantTypeProperty = {
  /**
   * current assistant
   */
  currentAssistant: null,

  /**
   * list of assistant
   */
  assistants: [],

  /**
   * should show instruction
   */
  instructionVisible: false,

  /**
   * edit tag visible
   */
  editTagVisible: false,

  /**
   * edit descipttion
   */

  updateDescriptionVisible: false,
};

/**
 *
 */
export const useAssistantPageStore = create<AssistantType>((set, get) => ({
  ...intialAssistant,
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
  onChangeCurrentAssistant: (assistant: Assistant) => {
    set({ currentAssistant: assistant });
  },

  /**
   *
   * @param ep
   */
  onChangeAssistants: (ep: Assistant[]) => {
    set({
      assistants: ep,
    });
  },

  /**
   *
   * @param assistant
   */
  reloadAssistant: (assistant: Assistant) => {
    get().onChangeAssistants([
      assistant,
      ...get().assistants.filter(at => assistant.getId() !== at.getId()),
    ]);
  },

  /**
   *
   */
  onClearCurrentAssistant: () => {
    set({
      currentAssistant: null,
    });
  },

  /**
   *
   * @param assistant
   */
  onAddAssistant: (assistant: Assistant) => {
    get().onReloadAssistant(assistant);
  },

  onReloadAssistant: (assistant: Assistant) => {
    get().onChangeCurrentAssistant(assistant);
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
   *
   * @param projectId
   * @param token
   * @param userId
   */
  onGetAllAssistant: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Assistant[]) => void,
  ) => {
    const afterGetAllAssistant = (
      err: ServiceError | null,
      gur: GetAllAssistantResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        get().onChangeAssistants(gur.getDataList());
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
          'Something went wrong while retrieving your assistants. Please refresh the page or try again later.',
        );
      }
    };

    GetAllAssistant(
      connectionConfig,
      get().page,
      get().pageSize,
      get().criteria,
      afterGetAllAssistant,
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
   * @param assistantProviderModelId
   * @param projectId
   * @param token
   * @param userId
   * @param onSuccess
   * @param onError
   */
  onGetAssistant: (
    assistantId: string,
    assistantProviderModelId: string | null,
    projectId: string,
    token: string,
    userId: string,
    onSuccess: (assistant: Assistant) => void,
    onError: (err: string) => void,
  ) => {
    const request = new GetAssistantRequest();
    const assistantDef = new AssistantDefinition();
    assistantDef.setAssistantid(assistantId);
    if (assistantProviderModelId)
      assistantDef.setVersion(assistantProviderModelId);
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
      .then(epmr => {
        if (epmr?.getSuccess()) {
          let assistant = epmr.getData();
          if (assistant) {
            onSuccess(assistant);
            return;
          }
        } else {
          let errorMessage =
            'Unable to get your assistant. please try again later.';
          const error = epmr?.getError();
          if (error) {
            errorMessage = error.getHumanmessage();
          }
          onError(errorMessage);
          return;
        }
      })
      .catch(err => {
        onError(
          'Something went wrong while retrieving your assistant. Please refresh the page or try again later.',
        );
      });
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
   *
   * @param is
   * @returns
   */
  onHideInstruction: () => {
    set({
      instructionVisible: false,
    });
  },

  /**
   *
   * @param ep
   * @param epm
   * @returns
   */
  onShowInstruction: (ep: Assistant) => {
    set({
      currentAssistant: ep,
      instructionVisible: true,
    });
  },

  /**
   *
   * @param ep
   */
  onShowUpdateDescription: (ep: Assistant) => {
    set({
      currentAssistant: ep,
      updateDescriptionVisible: true,
    });
  },

  /**
   *
   */
  onHideUpdateDescription: () => {
    set({
      updateDescriptionVisible: false,
    });
  },

  /**
   *
   * @param assistantId
   * @param name
   * @param description
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onUpdateAssistantDescription: (
    assistantId: string,
    name: string,
    description: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Assistant) => void,
  ) => {
    const afterUpdateAssistant = (
      err: ServiceError | null,
      gur: GetAssistantResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        let wf = gur.getData();
        if (wf) {
          get().onReloadAssistant(wf);
          onSuccess(wf);
        }
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError('Unable to update assistant, please try again later.');
      }
    };

    // when you have api then you can uncomment it
    UpdateAssistantDetail(
      connectionConfig,
      assistantId,
      name,
      description,
      afterUpdateAssistant,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
    );
  },

  /**
   *
   * @param ep
   */
  onShowEditTagVisible: (ep: Assistant) => {
    set({
      editTagVisible: true,
    });
  },

  /**
   *
   */
  onHideEditTagVisible: () => {
    set({
      editTagVisible: false,
    });
  },

  /**
   *
   * @param assistantId
   * @param projectId
   * @param tags
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onCreateAssistantTag: (
    assistantId: string,
    tags: string[],
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Assistant) => void,
  ) => {
    const afterCreateAssistantTag = (
      err: ServiceError | null,
      gur: GetAssistantResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        let assistant = gur.getData();
        if (assistant) {
          get().onReloadAssistant(assistant);
          onSuccess(assistant);
        }
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError('Unable to update assistant tag, please try again later.');
      }
    };

    CreateAssistantTag(
      connectionConfig,
      assistantId,
      tags,
      afterCreateAssistantTag,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
    );
  },

  /**
   * clear everything from the context
   * @returns
   */
  clear: () =>
    set(state => ({
      ...intialAssistant,
      ...initialPaginatedState,
    })),
}));

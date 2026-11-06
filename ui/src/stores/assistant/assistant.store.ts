import { ServiceError } from '@rapidaai/react';
import { Assistant, GetAssistantResponse } from '@rapidaai/react';

import { create } from 'zustand';
import { AssistantType, AssistantTypeProperty } from '@/types';
import {
  initialPaginated,
  initialPaginatedState,
} from '@/types/types.paginated';
import {
  createAssistantTags,
  getAssistantById,
  listAssistants,
  updateAssistantDescription,
} from '@/clients/assistant.client';

const intialAssistant: AssistantTypeProperty = {
  /**
   * current assistant
   */
  currentAssistant: null,

  currentAssistantStatus: 'idle',

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
    set({ currentAssistant: assistant, currentAssistantStatus: 'ready' });
  },

  onChangeCurrentAssistantStatus: status => {
    set({ currentAssistantStatus: status });
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
    let current = get().criteria.filter(
      x => !(x.key === k && x.logic === logic),
    );
    if (v) current.push({ key: k, value: v, logic: logic });
    set({
      page: 1,
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
      if (c.v) current.push({ key: c.k, value: c.v, logic: c.logic });
    });
    set({
      page: 1,
      criteria: current,
    });
  },

  removeCriteria: (k: string) => {
    set(state => ({
      page: 1,
      criteria: state.criteria.filter(criterion => criterion.key !== k),
    }));
  },

  setCriterias: (v: { k: string; v: string; logic: string }[]) => {
    const criteria = v
      .filter(c => c.v)
      .map(c => ({
        key: c.k,
        logic: c.logic,
        value: c.v,
      }));

    set(state => {
      const isUnchanged =
        state.page === 1 &&
        state.criteria.length === criteria.length &&
        state.criteria.every(
          (current, index) =>
            current.key === criteria[index].key &&
            current.logic === criteria[index].logic &&
            current.value === criteria[index].value,
        );

      return isUnchanged ? state : { page: 1, criteria };
    });
  },

  clearCriteria: () => {
    set({
      page: 1,
      criteria: [],
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
    listAssistants({
      page: get().page,
      pageSize: get().pageSize,
      criteria: get().criteria,
      auth: { projectId, token, userId },
    })
      .then(gur => {
        if (gur?.getSuccess()) {
          get().onChangeAssistants(gur.getDataList());
          let paginated = gur.getPaginated();
          if (paginated) {
            get().setTotalCount(paginated.getTotalitem());
          }
          onSuccess(gur.getDataList());
          return;
        }

        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError(
          'Something went wrong while retrieving your assistants. Please refresh the page or try again later.',
        );
      })
      .catch(() => {
        onError(
          'Something went wrong while retrieving your assistants. Please refresh the page or try again later.',
        );
      });
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
    getAssistantById({
      assistantId,
      assistantProviderModelId,
      auth: { projectId, token, userId },
    })
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

    updateAssistantDescription({
      assistantId,
      name,
      description,
      auth: { projectId, token, userId },
      callback: afterUpdateAssistant,
    });
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

    createAssistantTags({
      assistantId,
      tags,
      auth: { projectId, token, userId },
      callback: afterCreateAssistantTag,
    });
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

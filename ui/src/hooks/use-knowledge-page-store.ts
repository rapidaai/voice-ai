import {
  CreateKnowledgeTag,
  GetAllKnowledgeBases,
  UpdateKnowledgeDetail,
} from '@rapidaai/react';
import { GetAllKnowledgeResponse, GetKnowledgeResponse } from '@rapidaai/react';
import { KnowledgeType, KnowledgeTypeProperty } from '@/types/types.knowledge';
import { initialPaginated } from '@/types/types.paginated';
import { create } from 'zustand';
import { Knowledge } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

/**
 *
 */

const initialKnowledgeType: KnowledgeTypeProperty = {
  currentKnowledge: null,
  /**
   * list of endpoint where these will be part of it
   */
  knowledgeBases: [],

  /**
   * edit tag
   */
  editTagVisible: false,

  /**
   *
   */
  updateDescriptionVisible: false,
};
export const useKnowledgePageStore = create<KnowledgeType>((set, get) => ({
  ...initialPaginated,
  ...initialKnowledgeType,
  /**
   * current knowledge which will be targeted
   */

  onChangeCurrentKnowledge: (currentKnowledge: Knowledge) => {
    set({ currentKnowledge: currentKnowledge });
  },

  /**
   *
   */
  onClearCurrentKnowledge: () => {
    set({
      currentKnowledge: null,
    });
  },

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
  onChangeKnowledges: (kbs: Knowledge[]) => {
    set({
      knowledgeBases: kbs,
    });
  },

  /**
   *
   * @param endpoint
   */
  onReloadKnowledge: (knowledge: Knowledge) => {
    get().onChangeKnowledges([
      knowledge,
      ...get().knowledgeBases.filter(kn => knowledge.getId() !== kn.getId()),
    ]);
    get().onChangeCurrentKnowledge(knowledge);
  },

  /**
   *
   * @param endpoint
   */
  onAddKnowledge: (knowledgeBase: Knowledge) => {
    get().onReloadKnowledge(knowledgeBase);
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
  getAllKnowledge: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Knowledge[]) => void,
  ) => {
    const afterGetAllKnowledge = (
      err: ServiceError | null,
      gur: GetAllKnowledgeResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        get().onChangeKnowledges(gur.getDataList());
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
          'Something went wrong while retrieving your knowledges. Please refresh the page or try again later.',
        );
      }
    };

    GetAllKnowledgeBases(
      connectionConfig,
      get().page,
      get().pageSize,
      get().criteria,
      afterGetAllKnowledge,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
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
   *
   */
  onShowUpdateDescription: (ep: Knowledge) => {
    set({
      currentKnowledge: ep,
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
   * @param knowledgeId
   * @param name
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @param description
   */
  onUpdateKnowledgeDetail: (
    knowledgeId: string,
    name: string,
    description: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Knowledge) => void,
  ) => {
    const afterUpdateKnowledge = (
      err: ServiceError | null,
      gur: GetKnowledgeResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        let wf = gur.getData();
        if (wf) {
          get().onReloadKnowledge(wf);
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
    UpdateKnowledgeDetail(
      connectionConfig,
      knowledgeId,
      name,
      description,
      afterUpdateKnowledge,
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
  onShowEditTagVisible: (ep: Knowledge) => {
    set({
      currentKnowledge: ep,
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
   * @param endpointId
   * @param projectId
   * @param tags
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   */
  onEditKnowledgeTag: (
    knowledgeId: string,
    tags: string[],
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Knowledge) => void,
  ) => {
    const afterCreateKnowledgeTag = (
      err: ServiceError | null,
      gur: GetKnowledgeResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        let kn = gur.getData();
        if (kn) {
          get().onReloadKnowledge(kn);
          onSuccess(kn);
        }
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError('Unable to update endpoint tag, please try again later.');
      }
    };
    CreateKnowledgeTag(
      connectionConfig,
      knowledgeId,
      tags,
      afterCreateKnowledgeTag,
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
  clear: () => set({ ...initialKnowledgeType }, true),
}));

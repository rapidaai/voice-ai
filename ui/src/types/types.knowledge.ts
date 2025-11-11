import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { Knowledge } from '@rapidaai/react';

/**
 * knowledge context
 */

export type KnowledgeTypeProperty = {
  /**
   *
   */
  currentKnowledge: Knowledge | null;

  /**
   *
   */
  knowledgeBases: Knowledge[];

  /**
   * edit tag visible
   */
  editTagVisible: boolean;
  /**
   * edit descipt
   */

  updateDescriptionVisible: boolean;
};

//
//
export type KnowledgeAction = {
  /**
   *
   * @param ep
   * @returns
   */
  onShowEditTagVisible: (ep: Knowledge) => void;

  /**
   *
   * @returns
   */
  onHideEditTagVisible: () => void;

  /**
   *
   * @param ep
   * @returns
   */
  onShowUpdateDescription: (ep: Knowledge) => void;
  /**
   *
   * @returns
   */
  onHideUpdateDescription: () => void;
  /**
   *
   * @param knowledge
   * @returns
   */
  onChangeCurrentKnowledge: (knowledge: Knowledge) => void;
  /**
   *
   * @returns
   */
  onClearCurrentKnowledge: () => void;

  /**
   *
   * @param ep
   * @returns
   */
  onChangeKnowledges: (ep: Knowledge[]) => void;

  /**
   *
   * @param knowledge
   * @returns
   */
  onReloadKnowledge: (knowledge: Knowledge) => void;

  /**
   *
   * @param knowledge
   * @returns
   */
  onAddKnowledge: (knowledge: Knowledge) => void;
};

//
//
//
export type KnowledgeType = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getAllKnowledge: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Knowledge[]) => void,
  ) => void;

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
   * @returns
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
  ) => void;

  /**
   *
   * @param knowledgeId
   * @param tags
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  onEditKnowledgeTag: (
    knowledgeId: string,
    tags: string[],
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: Knowledge) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & PaginatedType &
  KnowledgeTypeProperty &
  KnowledgeAction &
  ColumnarType;

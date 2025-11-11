import { ProviderConfig } from '@/app/components/providers';

export type CreateKnowledgeTypeProperty = {
  /**
   *
   * @param string
   * @returns
   */
  description: string;

  /**
   *
   */
  providerModel: ProviderConfig;

  /**
   * endpoint name
   */
  name: string;

  /**
   * endpoint tags
   */
  tags: string[];
};

export type CreateKnowledgeTypeAction = {
  /**
   *
   * @param s
   * @returns
   */
  onChangeDescription: (s: string) => void;

  /**
   * on Change of model
   * @param md
   * @returns
   */
  onChangeProviderModel: (providerConfig: ProviderConfig) => void;

  /**
   *
   * @param i
   * @param v
   * @returns
   */
  onChangeProvider: (i: string, v: string) => void;
  /**
   * set name
   */

  onChangeName: (s: string) => void;

  /**
   *
   * @param s
   * @returns
   */
  onRemoveTag: (s: string) => void;

  /**
   *
   * @param s
   * @returns
   */
  onAddTag: (s: string) => void;
};
export type CreateKnowledgeType = {
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @param onSuccess
   * @param onError
   * @returns
   */
  onCreateKnowledge: (
    projectId: string,
    token: string,
    userId: string,
    onSuccess: (knowledge: any) => void,
    onError: (err: string) => void,
  ) => void;

  /**
   *
   * @returns
   */
  clear: () => void;
} & CreateKnowledgeTypeProperty &
  CreateKnowledgeTypeAction;

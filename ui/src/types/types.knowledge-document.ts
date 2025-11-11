import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { KnowledgeDocument } from '@rapidaai/react';

/**
 * assistant context
 */

export type KnowledgeDocumentProperty = {
  /**
   * list of assistant
   */
  documents: KnowledgeDocument[];
};

/**
 *
 */
export type KnowledgeDocumentType = {
  setKnowledgeDocuments: (dc: KnowledgeDocument[]) => void;
  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getAllKnowledgeDocument: (
    knowledgeId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: KnowledgeDocument[]) => void,
  ) => void;

  /**
   *
   * @param knowledgeId
   * @param knowledgeDocumentId
   * @param indexType
   * @param projectId
   * @param token
   * @param userId
   * @param onError
   * @param onSuccess
   * @returns
   */
  indexKnowledgeDocument: (
    knowledgeId: string,
    knowledgeDocumentId: string[],
    indexType: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: boolean) => void,
  ) => void;
  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & KnowledgeDocumentProperty &
  PaginatedType &
  ColumnarType;

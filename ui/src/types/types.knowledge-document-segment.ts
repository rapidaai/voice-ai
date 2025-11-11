import { ColumnarType } from './types.columnar';
import { PaginatedType } from './types.paginated';
import { KnowledgeDocument, KnowledgeDocumentSegment } from '@rapidaai/react';

/**
 * assistant context
 */

export type KnowledgeDocumentSegmentTypeProperty = {
  /**
   *
   */
  currentKnowledgeDocument: KnowledgeDocument | null;
  /**
   * list of assistant
   */
  knowledgeDocumentSegments: KnowledgeDocumentSegment[];
};

/**
 *
 */
export type KnowledgeDocumentSegmentType = {
  /**
   *
   * @param d
   * @returns
   */
  onChangeCurrentKnowledgeDocument: (d: KnowledgeDocument) => void;
  /**
   *
   * @param ep
   * @returns
   */
  setKnowledgeDocumentSegments: (ep: KnowledgeDocumentSegment[]) => void;

  /**
   *
   * @param projectId
   * @param token
   * @param userId
   * @returns
   */
  getAllKnowledgeDocumentSegment: (
    knowledge_id: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: KnowledgeDocumentSegment[]) => void,
  ) => void;

  /**
   * clear everything
   * @returns
   */
  clear: () => void;
} & KnowledgeDocumentSegmentTypeProperty &
  PaginatedType &
  ColumnarType;

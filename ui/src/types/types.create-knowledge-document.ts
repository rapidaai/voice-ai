import { Content } from '@rapidaai/react';
import { KnowledgeDocument } from '@rapidaai/react';
import {
  RapidaDocumentPreProcessing,
  RapidaDocumentSource,
  RapidaDocumentType,
} from '@/utils/rapida_document';

export type CreateKnowledgeDocumentProperty = {
  /**
   * importer Type
   * Mostly connectors and other object
   *
   * DOCUMENT_SOURCE_MANUAL_FILE = 0,
   * DOCUMENT_SOURCE_MANUAL_ZIP = 1,
   * DOCUMENT_SOURCE_MANUAL_URL = 2,
   */
  datasource: string;

  /**
   * document source
   *
   */
  documentSource: RapidaDocumentSource;

  /**
   * document Type
   */
  documentType: RapidaDocumentType;

  /**
   * Pre processing
   */
  preProcessing: RapidaDocumentPreProcessing;

  /**
   * separator
   */
  separator: string;

  /**
   * chunking size
   */
  maxChunkSize: number;

  /**
   * chunk overlap
   */
  chunkOverlap: number;

  /**
   *
   */
  knowledgeDocuments: Array<{
    file: Uint8Array;
    type: string;
    size: number;
    name: string;
  }>;

  /**
   *
   */
  knowledgeWebsiteUrl: string | null;
};
export type CreateKnowledgeDocumentAction = {
  /**
   *
   * @returns
   */
  onChangeDocumentType: (dt: RapidaDocumentType) => void;

  /**
   *
   * @param it
   * @returns
   */
  onChangeDatasource: (it: string) => void;

  /**
   *
   * @param it
   * @returns
   */
  onChangeDocumentSource: (it: RapidaDocumentSource) => void;

  /**
   *
   * @param pp
   * @returns
   */
  onChangePreProcessor: (pp: RapidaDocumentPreProcessing) => void;

  /**
   *
   * @param s
   * @returns
   */
  onChangeSeparator: (s: string) => void;

  /**
   *
   * @param co
   * @returns
   */
  onChangeMaxChunkSize: (co: string) => void;
  /**
   *
   * @param co
   * @returns
   */
  onChangeChunkOverlap: (co: string) => void;

  /**
   *
   * @param fl
   * @returns
   */
  onAddKnowledgeDocument: (fl: {
    file: Uint8Array;
    type: string;
    size: number;
    name: string;
  }) => void;

  onRemoveKnowledgeDocument: (name: string) => void;

  /**
   *
   * @param s
   * @returns
   */
  onChangeKnowledgeWebsite: (s: string) => void;
};

/**
 *
 */
export type CreateKnowledgeDocumentType = {
  /**
   *
   * @param knowledgeId
   * @param projectId
   * @param token
   * @param userId
   * @param onSuccess
   * @param onError
   * @returns
   */
  onCreateKnowledgeDocument: (
    knowledgeId: string,
    projectId: string,
    token: string,
    userId: string,
    onSuccess: (d: KnowledgeDocument[]) => void,
    onError: (e: string) => void,
  ) => void;

  /**
   *
   * @param knowledgeId
   * @param contents
   * @param projectId
   * @param token
   * @param userId
   * @param onSuccess
   * @param onError
   * @returns
   */
  onCreateDocument: (
    knowledgeId: string,
    documentSource: RapidaDocumentSource,
    datasource: string,
    contents: Array<Content>,
    projectId: string,
    token: string,
    userId: string,
    onSuccess: (d: KnowledgeDocument[]) => void,
    onError: (e: string) => void,
  ) => void;
  /**
   *
   * @returns
   */
  clear: () => void;
} & CreateKnowledgeDocumentProperty &
  CreateKnowledgeDocumentAction;

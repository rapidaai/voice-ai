import { ConnectionConfig, CreateKnowledgeDocument } from '@rapidaai/react';
import {
  CreateKnowledgeDocumentResponse,
  KnowledgeDocument,
} from '@rapidaai/react';
import {
  CreateKnowledgeDocumentProperty,
  CreateKnowledgeDocumentType,
} from '@/types/types.create-knowledge-document';
import { create } from 'zustand';
import { Content } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import {
  RapidaDocumentPreProcessing,
  RapidaDocumentSource,
  RapidaDocumentType,
} from '@/utils/rapida_document';
import { connectionConfig } from '@/configs';

const initialState: CreateKnowledgeDocumentProperty = {
  /**
   * defautl document type is unstructure
   */
  documentType: RapidaDocumentType.UNSTRUCTURE,

  /**
   * importer Type
   * Mostly connectors and other object
   */
  datasource: 'manual-file',

  /**
   *
   */
  documentSource: RapidaDocumentSource.MANUAL,

  /**
   * Pre processing
   */
  preProcessing: RapidaDocumentPreProcessing.AUTOMATIC,

  /**
   * separator
   */
  separator: '.',

  /**
   * chunking size
   */
  maxChunkSize: 500,

  /**
   * chunk overlap
   */
  chunkOverlap: 50,

  /**
   *
   */
  knowledgeDocuments: [],

  /**
   *
   */
  knowledgeWebsiteUrl: null,
};

export const useCreateKnowledgeDocumentPageStore =
  create<CreateKnowledgeDocumentType>((set, get) => ({
    ...initialState,

    /**
     *
     * @param it
     */
    onChangeDatasource: (it: string) => {
      set({
        datasource: it,
      });
    },

    onChangeDocumentType: (dt: RapidaDocumentType) => {
      set({
        documentType: dt,
      });
    },
    /**
     *
     * @param gt
     */
    onChangeDocumentSource: (gt: RapidaDocumentSource) => {
      set({
        documentSource: gt,
      });
    },

    /**
     *
     * @param pp
     * @returns
     */
    onChangePreProcessor: (pp: RapidaDocumentPreProcessing) => {
      set({
        preProcessing: pp,
      });
    },

    /**
     *
     * @param s
     * @returns
     */
    onChangeSeparator: (s: string) => {
      set({
        separator: s,
      });
    },

    /**
     *
     * @param co
     * @returns
     */
    onChangeMaxChunkSize: (co: string) => {
      if (!Number(co)) return;
      set({
        maxChunkSize: Number(co),
      });
    },

    /**
     *
     * @param co
     * @returns
     */
    onChangeChunkOverlap: (co: string) => {
      if (!Number(co)) return;
      set({
        chunkOverlap: Number(co),
      });
    },
    /**
     * clear everything from the context
     * @returns
     */

    onRemoveKnowledgeDocument: s => {
      let old = get().knowledgeDocuments.filter(x => x.name !== s);
      set({
        knowledgeDocuments: [...old],
      });
    },
    /**
     *
     * @param fl
     */
    onAddKnowledgeDocument: (fl: {
      file: Uint8Array;
      type: string;
      size: number;
      name: string;
    }) => {
      let old = get().knowledgeDocuments.filter(x => x.name !== fl.name);
      set({
        knowledgeDocuments: [...old, fl],
      });
    },

    onChangeKnowledgeWebsite: (s: string) => {
      set({
        knowledgeWebsiteUrl: s,
      });
    },

    /**
     *
     * @param knowledgeId
     * @param documentSource
     * @param datasource
     * @param Array
     * @param
     * @param Content
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
    ) => {
      const afterCreateKnowledgeDocument = (
        err: ServiceError | null,
        uvcr: CreateKnowledgeDocumentResponse | null,
      ) => {
        if (uvcr?.getSuccess()) {
          let kd = uvcr.getDataList();
          if (kd) {
            onSuccess(kd);
          }
        } else {
          const errorMessage =
            'Unable to upload the knowledge document. please try again later.';
          const error = uvcr?.getError();
          if (error) {
            onError(error.getHumanmessage());
            return;
          }
          onError(errorMessage);
        }
        return;
      };

      CreateKnowledgeDocument(
        connectionConfig,
        knowledgeId,
        documentSource,
        datasource,
        get().documentType,
        RapidaDocumentPreProcessing.AUTOMATIC,
        contents,
        get().separator,
        get().maxChunkSize,
        get().chunkOverlap,
        afterCreateKnowledgeDocument,
        ConnectionConfig.WithDebugger({
          authorization: token,
          userId: userId,
          projectId: projectId,
        }),
      );
    },

    /**
     *
     */
    onCreateKnowledgeDocument: (
      knowledgeId: string,
      projectId: string,
      token: string,
      userId: string,
      onSuccess: (d: KnowledgeDocument[]) => void,
      onError: (e: string) => void,
    ) => {
      const contents: Array<Content> = [];
      switch (get().documentSource) {
        default:
          onError('Please select requested document source.');
          return;
        case RapidaDocumentSource.MANUAL:
          switch (get().datasource) {
            case 'manual-file':
              const kd = get().knowledgeDocuments;
              if (kd.length === 0) {
                onError(
                  'Please select a document that can be used as knowledge document',
                );
                return;
              }
              kd.forEach(x => {
                const cntn = new Content();
                cntn.setContent(x.file);
                cntn.setName(x.name);
                cntn.setContenttype(x.type);
                contents.push(cntn);
              });
              break;
            case 'manual-url':
              const weburl = get().knowledgeWebsiteUrl;
              if (weburl === null) {
                onError(
                  'Please provide a public url that can be used as knowledge.',
                );
                return;
              }

              const cntn = new Content();
              cntn.setContent(new TextEncoder().encode(weburl));
              cntn.setName(weburl);
              cntn.setContentformat('url');
              cntn.setContenttype('text/html');
              contents.push(cntn);
              break;
          }
      }

      let preProcessing = RapidaDocumentPreProcessing.AUTOMATIC;
      if (get().preProcessing === RapidaDocumentPreProcessing.CUSTOM) {
        preProcessing = RapidaDocumentPreProcessing.CUSTOM;
        if (!(get().chunkOverlap > 0)) {
          onError('Please provide a valid chunk overlap for preprocessing.');
          return;
        }
        if (!(get().maxChunkSize > 0)) {
          onError('Please provide a valid chunk size for preprocessing.');
          return;
        }
        if (get().separator === '') {
          onError('Please provide a valid separator string for preprocessing.');
          return;
        }
      }

      const afterCreateKnowledgeDocument = (
        err: ServiceError | null,
        uvcr: CreateKnowledgeDocumentResponse | null,
      ) => {
        if (uvcr?.getSuccess()) {
          let kd = uvcr.getDataList();
          if (kd) {
            onSuccess(kd);
          }
        } else {
          const errorMessage =
            'Unable to upload the knowledge document. please try again later.';
          const error = uvcr?.getError();
          if (error) {
            onError(error.getHumanmessage());
            return;
          }
          onError(errorMessage);
        }
        return;
      };

      CreateKnowledgeDocument(
        connectionConfig,
        knowledgeId,
        get().documentSource,
        get().datasource,
        get().documentType,
        preProcessing,
        contents,
        get().separator,
        get().maxChunkSize,
        get().chunkOverlap,
        afterCreateKnowledgeDocument,
        ConnectionConfig.WithDebugger({
          authorization: token,
          userId: userId,
          projectId: projectId,
        }),
      );
    },
    clear: () => {
      set({ ...initialState }, false);
    },
  }));

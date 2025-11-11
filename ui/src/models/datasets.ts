import { Knowledge, Metadata } from '@rapidaai/react';

export enum RETRIEVE_METHOD {
  semantic = 'semantic',
  fullText = 'fullText',
  hybrid = 'hybrid',
  invertedIndex = 'invertedIndex',
}
export type RetrievalConfig = {
  searchMethod: RETRIEVE_METHOD;
  rerankingEnable: boolean;
  rerankerModelProvider?: string;
  rerankerModelProviderId?: string;
  rerankerModelOptions?: Metadata[];
  topK: number;
  scoreThreshold: number;
};

export const DEFAULT_RETRIVAL_CONFIG = {
  searchMethod: RETRIEVE_METHOD.semantic,
  rerankingEnable: false,
  topK: 5,
  scoreThreshold: 0.5,
};

export type Dataset = {
  knowledge: Knowledge;
  config: RetrievalConfig;
  active: boolean;
};

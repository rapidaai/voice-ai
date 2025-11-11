import { create } from 'zustand';
import { Knowledge } from '@rapidaai/react';
import { CreateKnowledgeType } from '@/types/types.create-knowledge';
import { CreateKnowledgeTypeProperty } from '../types/types.create-knowledge';
import { CreateKnowledge } from '@rapidaai/react';
import { CreateKnowledgeResponse } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { ProviderConfig } from '@/app/components/providers';
import {
  GetDefaultEmbeddingConfigIfInvalid,
  ValidateEmbeddingDefaultOptions,
} from '@/app/components/providers/embedding';
import { randomMeaningfullName } from '@/utils';
import { connectionConfig } from '@/configs';

/**
 *
 */

const initialState: CreateKnowledgeTypeProperty = {
  /**
   * description of endpoint provider model
   */
  description: '',

  /**
   * name of the endpoint
   */
  name: randomMeaningfullName('knowledge'),
  /**
   *
   */
  providerModel: {
    provider: 'openai',
    providerId: '1987967168452493312',
    parameters: GetDefaultEmbeddingConfigIfInvalid('openai', []),
  },

  /**
   * list of tags for endpoint
   */
  tags: [],
};

export const useCreateKnowledgePageStore = create<CreateKnowledgeType>(
  (set, get) => ({
    ...initialState,

    onChangeDescription: (st: string) => {
      set({
        description: st,
      });
    },

    /**
     *
     * on change of name of endpoint
     * @param st
     */
    onChangeName: (st: string) => {
      set({
        name: st,
      });
    },

    /**
     *
     * @param m
     */
    onChangeProviderModel: (m: ProviderConfig) => {
      set({
        providerModel: m,
      });
    },

    /**
     *
     */
    onChangeProvider: (i: string, v: string) => {
      set({
        providerModel: {
          providerId: i,
          provider: v,
          parameters: [],
        },
      });
    },

    /**
     * adding new tags of endpoint
     * @param s
     */
    onAddTag: (s: string) => {
      if (s.trim() !== '') {
        let oldTags = get().tags;
        const index = oldTags.indexOf(s, 0);
        if (index > -1) {
          return;
        }
        const all = [...oldTags, s];
        set({ tags: all });
      }
    },

    /**
     * remove the tags for endpoint
     */
    onRemoveTag: (s: string) => {
      set({
        tags: get().tags.filter(x => x !== s),
      });
    },

    /**
     *
     * @param projectId
     * @param token
     * @param userId
     * @param onSuccess
     * @param onError
     */
    onCreateKnowledge: async (
      projectId: string,
      token: string,
      userId: string,
      onSuccess: (knowledge: Knowledge) => void,
      onError: (err: string) => void,
    ) => {
      // validations
      let _providerModel = get().providerModel;
      if (!_providerModel) {
        onError('Please select the embedding models.');
        return;
      }

      let err = ValidateEmbeddingDefaultOptions(
        _providerModel.provider,
        _providerModel.parameters,
      );
      if (err) {
        onError(err);
        return;
      }
      let _name = get().name;
      if (!_name) {
        onError('Please enter the name of knowledge base.');
        return;
      }
      let _description = get().description;
      if (!_description) {
        onError('Please enter the description of knowledge base.');
        return;
      }

      let _tags = get().tags;
      CreateKnowledge(
        connectionConfig,
        _providerModel,
        _name,
        _description,
        _tags,
        {
          authorization: token,
          'x-auth-id': userId,
          'x-project-id': projectId,
        },
        (err: ServiceError | null, car: CreateKnowledgeResponse | null) => {
          if (car?.getSuccess()) {
            let assistant = car.getData();
            if (assistant) onSuccess(assistant);
          } else {
            const errorMessage =
              'Unable to create knowledge. please try again later.';
            const error = car?.getError();
            if (error) {
              onError(error.getHumanmessage());
              return;
            }
            onError(errorMessage);
            return;
          }
        },
      );
    },

    /**
     * clear everything from the context
     * @returns
     */
    clear: () => {
      set({ ...initialState }, false);
    },
  }),
);

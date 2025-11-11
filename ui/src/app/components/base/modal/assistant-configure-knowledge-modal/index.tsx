import { Card } from '@/app/components/base/cards';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { KnowledgeDropdown } from '@/app/components/Dropdown/knowledge-dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import CheckboxCard from '@/app/components/Form/checkbox-card';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { SwitchWithLabel } from '@/app/components/Form/Switch';
import { HybridSearchIcon } from '@/app/components/Icon/hybrid-search';
import { TextSearchIcon } from '@/app/components/Icon/text-search';
import { VectorSearchIcon } from '@/app/components/Icon/vector-search';
import { InputHelper } from '@/app/components/input-helper';
import {
  GetDefaultRerankerConfigIfInvalid,
  RerankerProvider,
} from '@/app/components/providers/reranker';
import { Tooltip } from '@/app/components/Tooltip';
import { RETRIEVE_METHOD } from '@/models/datasets';
import { cn } from '@/styles/media';
import { InfoIcon } from 'lucide-react';
import { FC, useEffect, useState } from 'react';
interface ConfigureAssistantKnowledgeDialogProps extends ModalProps {
  initialConfig: {
    knowledgeName: string;
    knowledgeDescription: string;
    knowledgeId: string;
    searchType: RETRIEVE_METHOD;
    topK: number;
    scoreThreshold: number;
    rerankingEnable: boolean;
    providerModel: {
      provider: string;
      providerId: string;
      parameters: any;
    };
  } | null;
  onChange: (config: {
    knowledgeName: string;
    knowledgeDescription: string;
    knowledgeId: string;
    searchType: RETRIEVE_METHOD;
    topK: number;
    scoreThreshold: number;
    rerankingEnable: boolean;
    providerModel: {
      provider: string;
      providerId: string;
      parameters: any;
    };
  }) => void;
  onValidateConfig?: (config: {
    knowledgeName: string;
    knowledgeDescription: string;
    knowledgeId: string;
    searchType: RETRIEVE_METHOD;
    topK: number;
    scoreThreshold: number;
    rerankingEnable: boolean;
    providerModel: {
      provider: string;
      providerId: string;
      parameters: any;
    };
  }) => string | null; // Return error message or null if valid
}
export const ConfigureAssistantKnowledgeDialog: FC<
  ConfigureAssistantKnowledgeDialogProps
> = ({
  initialConfig,
  onChange,
  modalOpen,
  setModalOpen,
  onValidateConfig,
}) => {
  const [knowledgeId, setKnowledgeId] = useState('');
  const [knowledgeName, setKnowledgeName] = useState('');
  const [knowledgeDescription, setKnowledgeDescription] = useState('');
  const [searchType, setSearchType] = useState(RETRIEVE_METHOD.hybrid);
  const [topK, setTopK] = useState(3);
  const [scoreThreshold, setScoreThreshold] = useState(0.5);
  const [rerankingEnable, setRerankingEnable] = useState(false);
  const [providerModel, setProviderModel] = useState({
    provider: 'cohere',
    providerId: '1987967168435716096',
    parameters: GetDefaultRerankerConfigIfInvalid('cohere', []),
  });

  const [errorMessage, setErrorMessage] = useState('');

  const resetState = () => {
    setKnowledgeId('');
    setKnowledgeName('');
    setKnowledgeDescription('');
    setSearchType(RETRIEVE_METHOD.hybrid);
    setTopK(3);
    setScoreThreshold(0.5);
    setRerankingEnable(false);
    setProviderModel({
      provider: 'cohere',
      providerId: '1987967168435716096',
      parameters: GetDefaultRerankerConfigIfInvalid('cohere', []),
    });
    setErrorMessage('');
  };

  useEffect(() => {
    if (modalOpen && initialConfig) {
      setKnowledgeId(initialConfig.knowledgeId);
      setKnowledgeName(initialConfig.knowledgeName);
      setKnowledgeDescription(initialConfig.knowledgeDescription);
      setSearchType(initialConfig.searchType);
      setTopK(initialConfig.topK);
      setScoreThreshold(initialConfig.scoreThreshold);
      setRerankingEnable(initialConfig.rerankingEnable);
      setProviderModel(initialConfig.providerModel);
    } else if (!modalOpen) {
      resetState();
    }
  }, [initialConfig, modalOpen]);

  const validateForm = () => {
    if (!knowledgeId.trim()) {
      setErrorMessage('Please select a valid knowledge.');
      return false;
    }
    if (topK < 1 || topK > 20) {
      setErrorMessage(
        'Please provide a Top K value between 1 and 20. This determines the number of most relevant results to retrieve.',
      );
      return false;
    }
    if (scoreThreshold < 0 || scoreThreshold > 1) {
      setErrorMessage(
        'Please provide a Score Threshold between 0 and 1. This value filters results based on their relevance score.',
      );
      return false;
    }
    setErrorMessage('');
    return true;
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateForm()) return;

    const updatedConfig = {
      knowledgeName,
      knowledgeDescription,
      knowledgeId,
      searchType,
      topK,
      scoreThreshold,
      rerankingEnable,
      providerModel,
    };
    if (onValidateConfig) {
      const parentError = onValidateConfig(updatedConfig);
      if (parentError) {
        setErrorMessage(parentError);
        return;
      }
    }

    onChange(updatedConfig);
  };

  return (
    <GenericModal
      className="flex"
      modalOpen={modalOpen}
      setModalOpen={setModalOpen}
    >
      <ModalFitHeightBlock className="w-[1000px]">
        <ModalHeader
          onClose={() => {
            setModalOpen(false);
          }}
        >
          <ModalTitleBlock>Connect Assistant Knowledge</ModalTitleBlock>
        </ModalHeader>
        <ModalBody className="overflow-auto max-h-[80dvh]">
          <KnowledgeDropdown
            className="bg-white"
            currentKnowledge={knowledgeId}
            onChangeKnowledge={k => {
              if (k) {
                setKnowledgeDescription(k.getDescription());
                setKnowledgeName(k.getName());
                setKnowledgeId(k.getId());
              }
            }}
          />
          <FieldSet>
            <FormLabel>Retrieval setting</FormLabel>
            <div className="grid grid-cols-3 gap-3">
              <CheckboxCard
                type="radio"
                name="search-type"
                id="hybrid-search-type"
                value="hybrid"
                checked={searchType === RETRIEVE_METHOD.hybrid}
                onChange={() => setSearchType(RETRIEVE_METHOD.hybrid)}
              >
                <Card className="p-3 flex flex-row space-x-3 bg-white">
                  <div className="rounded-[2px] flex items-center justify-center bg-blue-200/30 dark:bg-blue-200/10 shrink-0 h-10 w-10">
                    <HybridSearchIcon className="text-blue-600" />
                  </div>
                  <div className="flex flex-col">
                    <span className="font-medium text-[14px]">
                      Hybrid Search
                    </span>
                    <span className="text-sm opacity-80">
                      Execute full-text search and vector searches
                      simultaneously, re-rank to select the best match for the
                      user's query.
                    </span>
                  </div>
                </Card>
              </CheckboxCard>
              <CheckboxCard
                type="radio"
                name="search-type"
                id="vector-search-type"
                value="semantic"
                checked={searchType === RETRIEVE_METHOD.semantic}
                onChange={() => setSearchType(RETRIEVE_METHOD.semantic)}
              >
                <Card className="p-3 flex flex-row space-x-3 bg-white">
                  <div className="rounded-[2px] flex items-center justify-center bg-blue-200/30 dark:bg-blue-200/10 shrink-0 h-10 w-10">
                    <VectorSearchIcon className="text-blue-600" />
                  </div>
                  <div className="flex flex-col">
                    <span className="font-medium text-[14px]">
                      Semantic Search
                    </span>
                    <span className="text-sm opacity-80">
                      Generate query embeddings and search for the text chunk
                      most similar to its vector representation.
                    </span>
                  </div>
                </Card>
              </CheckboxCard>
              <CheckboxCard
                type="radio"
                name="search-type"
                id="text-search-type"
                value="text"
                checked={searchType === RETRIEVE_METHOD.fullText}
                onChange={() => setSearchType(RETRIEVE_METHOD.fullText)}
              >
                <Card className="p-3 flex flex-row space-x-3 bg-white">
                  <div className="rounded-[2px] flex items-center justify-center bg-blue-200/30 dark:bg-blue-200/10 shrink-0 h-10 w-10">
                    <TextSearchIcon className="text-blue-600" />
                  </div>
                  <div className="flex flex-col">
                    <span className="font-medium text-[14px]">
                      Full Text Search
                    </span>
                    <span className="text-sm opacity-80">
                      Index all terms in the document, allowing users to search
                      any term and retrieve relevant text chunk containing those
                      terms.
                    </span>
                  </div>
                </Card>
              </CheckboxCard>
            </div>
          </FieldSet>
          <div className="grid grid-cols-2 w-full gap-4">
            <FieldSet className="flex justify-between">
              <FormLabel htmlFor="top_k">
                Top K
                <Tooltip icon={<InfoIcon className="w-4 h-4 ml-1" />}>
                  <p className={cn('font-normal text-sm p-1 w-64')}>
                    Used to filter chunks that are most similar to user
                    questions. The system will also dynamically adjust the value
                    of Top K, according to max_tokens of the selected model.
                  </p>
                </Tooltip>
              </FormLabel>
              <div className="flex justify-between items-center space-x-2">
                <Slider
                  min={1}
                  max={10}
                  step={1}
                  value={topK}
                  onSlide={value => setTopK(value)}
                />
                <Input
                  id="top_k"
                  className={cn(
                    'py-0 px-1 tabular-nums border w-10 h-6 text-xs',
                  )}
                  min={0}
                  max={10}
                  type="number"
                  value={topK}
                  onChange={e => setTopK(Number(e.target.value))}
                />
              </div>
            </FieldSet>
            <FieldSet className="flex justify-between">
              <FormLabel htmlFor="score_threshold">
                Score Threshold
                <Tooltip icon={<InfoIcon className="w-4 h-4 ml-1" />}>
                  <p className={cn('font-normal text-sm p-1 w-64')}>
                    Used to filter chunks that are most similar to user
                    questions. The system will also dynamically adjust the value
                    of Top K, according to max_tokens of the selected model.
                  </p>
                </Tooltip>
              </FormLabel>
              <div className="flex justify-between items-center space-x-2">
                <Slider
                  min={0}
                  max={1}
                  step={0.1}
                  onSlide={value => setScoreThreshold(value)}
                  value={scoreThreshold}
                />
                <Input
                  id="score_threshold"
                  className={cn(
                    'py-0 px-1 tabular-nums border w-10 h-6 text-xs',
                  )}
                  min={0}
                  max={1}
                  type="number"
                  step=".01"
                  value={scoreThreshold}
                  onChange={e => setScoreThreshold(Number(e.target.value))}
                />
              </div>
            </FieldSet>
          </div>
          <FieldSet>
            <SwitchWithLabel
              enable={rerankingEnable}
              setEnable={setRerankingEnable}
              label="Enable reranking"
              className="bg-light-background"
            ></SwitchWithLabel>
            <InputHelper>
              Reranking improves the relevance of retrieved information by
              re-scoring and reordering the initial search results.
            </InputHelper>
          </FieldSet>
          <FieldSet>
            <RerankerProvider
              inputClass="bg-white"
              disabled={!rerankingEnable}
              config={providerModel}
              onChangeConfig={setProviderModel}
              onChangeProvider={() => {}}
            />
          </FieldSet>
        </ModalBody>
        <ModalFooter errorMessage={errorMessage}>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => {
              setModalOpen(false);
            }}
          >
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={onSubmit}
          >
            Connect knowledge
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};

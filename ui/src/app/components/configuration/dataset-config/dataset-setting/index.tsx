import { useEffect, useState, type FC } from 'react';
import { Input } from '@/app/components/Form/Input';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Tooltip } from '@/app/components/Tooltip';
import { InfoIcon } from '@/app/components/Icon/Info';
import CheckboxCard from '@/app/components/Form/checkbox-card';

import { VectorSearchIcon } from '@/app/components/Icon/vector-search';
import { Switch } from '@/app/components/Form/Switch';
import { Slider } from '@/app/components/Form/Slider';
import { Dataset } from '@/models/datasets';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { cn } from '@/styles/media';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { Card } from '@/app/components/base/cards';
import { HybridSearchIcon } from '@/app/components/Icon/hybrid-search';
import { TextSearchIcon } from '@/app/components/Icon/text-search';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { FormLabel } from '@/app/components/form-label';
import { InputGroup } from '@/app/components/input-group';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { RETRIEVE_METHOD } from '@/models/datasets';

type DatasetSettingProps = {
  isShow: boolean;
  onClose: () => void;
  readonly?: boolean;
  currentDataset: Dataset;
  onSave: (newDataset: Dataset) => void;
};

export const DatasetSetting: FC<DatasetSettingProps> = ({
  isShow,
  onClose,
  currentDataset,
  readonly,
  onSave,
}) => {
  const [cd, setCurrentDataset] = useState<Dataset>({ ...currentDataset });
  const [message, setMessage] = useState('');
  useEffect(() => {
    setCurrentDataset(currentDataset);
  }, [currentDataset]);

  const handleSearchMethodChange = (method: RETRIEVE_METHOD) => {
    setCurrentDataset({
      ...cd,
      config: { ...cd.config, searchMethod: method },
    });
  };

  const handleRerankingEnableChange = (enable: boolean) => {
    setCurrentDataset(prev => ({
      ...prev,
      config: { ...prev.config, rerankingEnable: enable },
    }));
  };

  const handleTopKChange = (value: number) => {
    setCurrentDataset(prev => ({
      ...prev,
      config: { ...prev.config, topK: value },
    }));
  };

  const handleScoreThresholdChange = (value: number) => {
    setCurrentDataset(prev => ({
      ...prev,
      config: { ...prev.config, scoreThreshold: value },
    }));
  };

  const handleModelChange = (
    rerankerModel: string,
    rerankerProviderId: string,
    rerankerProvider: string,
  ) => {
    setCurrentDataset(prev => ({
      ...prev,
      config: { ...prev.config, rerankerProvider },
    }));
  };

  //   useEffect(() => {
  //     if (defaultModel) handleModelChange(defaultModel);
  //   }, [defaultModel?.getId()]);

  const onConfigure = () => {
    if (
      cd.config.rerankingEnable &&
      !cd.config.rerankerModelProvider &&
      !cd.config.rerankerModelProviderId
    ) {
      setMessage('Please select one of the reranker model.');
      return;
    }
    onSave(cd);
  };
  return (
    <GenericModal modalOpen={isShow} setModalOpen={onClose}>
      <ModalFitHeightBlock>
        <ModalHeader onClose={onClose}>
          <ModalTitleBlock>Configure knowledge retrieval</ModalTitleBlock>
        </ModalHeader>
        <ModalBody className="space-y-0">
          <InputGroup
            initiallyExpanded
            className="px-0 -mx-4"
            title="Retrieval Document"
          >
            <div className="flex flex-col gap-4 p-6">
              <FieldSet>
                <FormLabel htmlFor="endpoint_name">
                  <span className="text-sm">Retrieval setting</span>
                  <Tooltip
                    icon={
                      <InfoIcon className="w-4 h-4 mt-[2px] ml-0.5 dark:text-gray-400" />
                    }
                  >
                    <p className={cn('font-normal text-sm p-1 w-64')}>
                      Give a name that you can use to identify the endpoint
                      later.
                    </p>
                  </Tooltip>
                </FormLabel>
                <div className="grid gap-3">
                  <CheckboxCard
                    type="radio"
                    name="search-type"
                    id="hybrid-search-type"
                    value="hybrid"
                    disabled={readonly}
                    checked={cd.config.searchMethod === RETRIEVE_METHOD.hybrid}
                    onChange={() =>
                      handleSearchMethodChange(RETRIEVE_METHOD.hybrid)
                    }
                  >
                    <Card className="p-3 flex flex-row space-x-3">
                      <div className="rounded-[2px] flex items-center justify-center bg-blue-200/30 dark:bg-blue-200/10 shrink-0 h-10 w-10">
                        <HybridSearchIcon className="text-blue-600" />
                      </div>
                      <div className="flex flex-col">
                        <span className="font-medium text-[14px]">
                          Hybrid Search
                        </span>
                        <span className="text-sm opacity-80">
                          Execute full-text search and vector searches
                          simultaneously, re-rank to select the best match for
                          the user's query.
                          {/* Configuration of the Rerank model API is
                      necessary. */}
                        </span>
                      </div>
                    </Card>
                  </CheckboxCard>
                  <CheckboxCard
                    type="radio"
                    name="search-type"
                    id="vector-search-type"
                    value="semantic"
                    disabled={readonly}
                    checked={
                      cd.config.searchMethod === RETRIEVE_METHOD.semantic
                    }
                    onChange={() =>
                      handleSearchMethodChange(RETRIEVE_METHOD.semantic)
                    }
                  >
                    <Card className="p-3 flex flex-row space-x-3">
                      <div className="rounded-[2px] flex items-center justify-center bg-blue-200/30 dark:bg-blue-200/10 shrink-0 h-10 w-10">
                        <VectorSearchIcon className="text-blue-600" />
                      </div>
                      <div className="flex flex-col">
                        <span className="font-medium text-[14px]">
                          Semantic Search
                        </span>
                        <span className="text-sm opacity-80">
                          Generate query embeddings and search for the text
                          chunk most similar to its vector representation.
                        </span>
                      </div>
                    </Card>
                  </CheckboxCard>
                  <CheckboxCard
                    type="radio"
                    name="search-type"
                    id="text-search-type"
                    value="text"
                    disabled={readonly}
                    checked={
                      cd.config.searchMethod === RETRIEVE_METHOD.fullText
                    }
                    onChange={() =>
                      handleSearchMethodChange(RETRIEVE_METHOD.fullText)
                    }
                  >
                    <Card className="p-3 flex flex-row space-x-3">
                      <div className="rounded-[2px] flex items-center justify-center bg-blue-200/30 dark:bg-blue-200/10 shrink-0 h-10 w-10">
                        <TextSearchIcon className="text-blue-600" />
                      </div>
                      <div className="flex flex-col">
                        <span className="font-medium text-[14px]">
                          Full Text Search
                        </span>
                        <span className="text-sm opacity-80">
                          Index all terms in the document, allowing users to
                          search any term and retrieve relevant text chunk
                          containing those terms.
                        </span>
                      </div>
                    </Card>
                  </CheckboxCard>
                </div>
              </FieldSet>
              <div className="grid grid-cols-2 w-full gap-4">
                <div className="space-y-2 col-span-1">
                  <FieldSet className="flex justify-between">
                    <FormLabel htmlFor="top_k">
                      Top K
                      <Tooltip icon={<InfoIcon className="w-4 h-4 ml-1" />}>
                        <p className={cn('font-normal text-sm p-1 w-64')}>
                          Used to filter chunks that are most similar to user
                          questions. The system will also dynamically adjust the
                          value of Top K, according to max_tokens of the
                          selected model.
                        </p>
                      </Tooltip>
                    </FormLabel>
                    <Input
                      id="top_k"
                      className={cn(
                        'py-0 px-1 tabular-nums border w-10 h-6 text-xs',
                      )}
                      min={0}
                      max={10}
                      type="number"
                      onChange={e => handleTopKChange(e.target.valueAsNumber)}
                      value={cd.config.topK}
                    />
                  </FieldSet>
                  <Slider
                    min={1}
                    max={10}
                    step={1}
                    onSlide={e => handleTopKChange(e)}
                    value={cd.config.topK}
                  />
                </div>
                <div className="space-y-2 col-span-1">
                  <FieldSet className="flex justify-between">
                    <FormLabel htmlFor="score_threshold">
                      Score Threshold
                      <Tooltip icon={<InfoIcon className="w-4 h-4 ml-1" />}>
                        <p className={cn('font-normal text-sm p-1 w-64')}>
                          Used to filter chunks that are most similar to user
                          questions. The system will also dynamically adjust the
                          value of Top K, according to max_tokens of the
                          selected model.
                        </p>
                      </Tooltip>
                    </FormLabel>
                    <Input
                      id="score_threshold"
                      className={cn(
                        'py-0 px-1 tabular-nums border w-10 h-6 text-xs',
                      )}
                      min={0}
                      max={1}
                      type="number"
                      onChange={e =>
                        handleScoreThresholdChange(e.target.valueAsNumber)
                      }
                      value={cd.config.scoreThreshold}
                    />
                  </FieldSet>
                  <Slider
                    min={0}
                    max={1}
                    step={0.1}
                    onSlide={e => handleScoreThresholdChange(e)}
                    value={cd.config.scoreThreshold}
                  />
                </div>
              </div>
            </div>
          </InputGroup>
          <InputGroup className="px-0 -mx-4" title="Reranking Document">
            <div className="flex flex-col gap-4 p-6">
              <FieldSet className="justify-between flex">
                <FormLabel>Reranking Enable</FormLabel>
                <Switch
                  enable={cd.config.rerankingEnable}
                  setEnable={handleRerankingEnableChange}
                />
              </FieldSet>
              {/* <FieldSet>
                <FormLabel htmlFor="Rerank Model">Rerank Model</FormLabel>
                <ProviderWithModelDropdown
                  allValue={rerankerProviderModels}
                  currentValue={defaultModel}
                  labelClassName="text-sm"
                  activeLabelClassName="text-sm"
                  setValue={handleModelChange}
                  disable={!cd.config.rerankingEnable}
                  placement={'top'}
                />
              </FieldSet> */}
            </div>
          </InputGroup>
          <ErrorMessage message={message} />
        </ModalBody>

        <ModalFooter className="sticky bottom-0">
          <ICancelButton className="rounded-[2px] px-8" onClick={onClose}>
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="rounded-[2px] px-8"
            type="button"
            onClick={onConfigure}
          >
            Configure knowledge
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};

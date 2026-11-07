import { Card } from '@/app/components/ui/primitives';
import { PageActionButtonBlock } from '@/app/components/layout/blocks/page-action-button-block';
import { KnowledgeDropdown } from '@/app/components/domain/dropdowns/knowledge-dropdown';
import { FormLabel } from '@/app/components/ui/primitives';
import { PrimaryButton, GhostButton } from '@/app/components/ui/primitives';
import { CheckboxCard } from '@/app/components/ui/primitives';
import { FieldSet } from '@/app/components/ui/primitives';
import { Input } from '@/app/components/ui/primitives';
import { Slider } from '@/app/components/ui/primitives';
import { SwitchWithLabel } from '@/app/components/ui/primitives';
import { InputHelper } from '@/app/components/ui/primitives';
import {
  GetDefaultRerankerConfigIfInvalid,
  RerankerProvider,
} from '@/app/components/domain/providers/reranker';
import { Tooltip } from '@/app/components/ui/primitives';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useRapidaStore } from '@/stores/app';
import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { RETRIEVE_METHOD } from '@/models/datasets';
import { cn } from '@/utils';
import { retrieveMethodFromString } from '@/utils';
import { DataBase, Information, ModelAlt, Search } from '@carbon/icons-react';
import { FC, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { useParams } from 'react-router-dom';
import {
  getAssistantKnowledgeLinkById,
  updateAssistantKnowledgeLink,
} from '@/clients/assistant.client';

export const UpdateKnowledge: FC<{ assistantId: string }> = ({
  assistantId,
}) => {
  const navigator = useGlobalNavigation();
  const { assistantKnowledgeId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { showLoader, hideLoader } = useRapidaStore();

  const [knowledgeId, setKnowledgeId] = useState('');
  const [searchType, setSearchType] = useState(RETRIEVE_METHOD.hybrid);
  const [topK, setTopK] = useState(5);
  const [scoreThreshold, setScoreThreshold] = useState(0.5);
  const [rerankingEnable, setRerankingEnable] = useState(false);
  const [providerModel, setProviderModel] = useState({
    provider: 'cohere',
    providerId: '1987967168435716096',
    parameters: GetDefaultRerankerConfigIfInvalid('cohere', []),
  });
  const [errorMessage, setErrorMessage] = useState('');
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});

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

  useEffect(() => {
    showLoader();
    getAssistantKnowledgeLinkById({
      assistantId,
      assistantKnowledgeId: assistantKnowledgeId!,
      auth: { projectId, token, userId: authId },
      callback: (err, res) => {
        hideLoader();
        if (err) {
          toast.error('Unable to assistant knowledge, please try again later.');
          return;
        }
        const aK = res?.getData();
        if (aK) {
          setKnowledgeId(aK.getKnowledgeid());
          setSearchType(retrieveMethodFromString(aK.getRetrievalmethod()));
          setTopK(aK.getTopk());
          setScoreThreshold(aK.getScorethreshold());
          setRerankingEnable(aK.getRerankerenable());
          if (aK.getRerankerenable()) {
            setProviderModel({
              providerId: aK.getRerankermodelproviderid(),
              provider: aK.getRerankermodelprovidername(),
              parameters: aK.getAssistantknowledgererankeroptionsList(),
            });
          }
        }
      },
    });
  }, [assistantId, assistantKnowledgeId, authId, token, projectId]);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validateForm()) return;

    try {
      updateAssistantKnowledgeLink({
        assistantKnowledgeId: assistantKnowledgeId!,
        assistantId,
        knowledgeId,
        retrievalOptions: {
          searchMethod: searchType,
          topK: topK,
          scoreThreshold: scoreThreshold,
          rerankingEnable: false,
        },
        auth: { projectId, token, userId: authId },
        callback: (err, response) => {
          if (err) {
            setErrorMessage(
              'Unable to update assistant knowledge, please check and try again.',
            );
          }
          if (response?.getSuccess()) {
            toast.success(`Assistant's knowledge updated successfully`);
            navigator.goToConfigureAssistantKnowledge(assistantId);
          } else {
            if (response?.getError()) {
              let err = response.getError();
              const message = err?.getHumanmessage();
              if (message) {
                setErrorMessage(message);
                return;
              }
              setErrorMessage(
                'Unable to update assistant knowledge, please check and try again.',
              );
              return;
            }
            setErrorMessage(
              'Unable to update assistant knowledge, please check and try again.',
            );
          }
        },
      });
    } catch (error) {
      setErrorMessage('Failed to configure webhook. Please try again.');
      console.error('Error configuring webhook:', error);
    }
  };

  return (
    <form
      method="POST"
      className="relative flex flex-col flex-1"
      onSubmit={onSubmit}
    >
      <ConfirmDialogComponent />
      <div className="overflow-auto flex flex-col flex-1 pb-20">
        <div className=" bg-surface ">
          <div
            className={cn(
              'px-6 py-6 flex flex-col gap-8 pl-8 w-full max-w-6xl',
            )}
          >
            <KnowledgeDropdown
              currentKnowledge={knowledgeId}
              onChangeKnowledge={kn => {
                if (kn) setKnowledgeId(kn.getId());
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
                  <Card className="p-3 flex flex-row space-x-3 bg-light-background">
                    <div className="rounded-[2px] flex items-center justify-center bg-support-info/30 shrink-0 h-10 w-10">
                      <DataBase className="text-support-info" />
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
                  <Card className="p-3 flex flex-row space-x-3 bg-light-background">
                    <div className="rounded-[2px] flex items-center justify-center bg-support-info/30 shrink-0 h-10 w-10">
                      <ModelAlt className="text-support-info" />
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
                  <Card className="p-3 flex flex-row space-x-3 bg-light-background">
                    <div className="rounded-[2px] flex items-center justify-center bg-support-info/30 shrink-0 h-10 w-10">
                      <Search className="text-support-info" />
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
              <FieldSet className="flex justify-between">
                <FormLabel htmlFor="top_k">
                  Top K
                  <Tooltip icon={<Information className="w-4 h-4 ml-1" />}>
                    <p className={cn('font-normal text-sm p-1 w-64')}>
                      Used to filter chunks that are most similar to user
                      questions. The system will also dynamically adjust the
                      value of Top K, according to max_tokens of the selected
                      model.
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
                  <Tooltip icon={<Information className="w-4 h-4 ml-1" />}>
                    <p className={cn('font-normal text-sm p-1 w-64')}>
                      Used to filter chunks that are most similar to user
                      questions. The system will also dynamically adjust the
                      value of Top K, according to max_tokens of the selected
                      model.
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
                provider={providerModel.provider}
                parameters={providerModel.parameters}
                onChangeParameter={() => {}}
                onChangeProvider={() => {}}
              />
            </FieldSet>
          </div>
        </div>
      </div>
      <PageActionButtonBlock errorMessage={errorMessage}>
        <GhostButton
          size="md"
          onClick={() => showDialog(navigator.goBack)}
          type="button"
        >
          Cancel
        </GhostButton>
        <PrimaryButton size="md" type="submit">
          Update knowledge config
        </PrimaryButton>
      </PageActionButtonBlock>
    </form>
  );
};

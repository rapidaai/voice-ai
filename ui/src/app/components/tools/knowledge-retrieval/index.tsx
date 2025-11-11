import { Knowledge, Metadata } from '@rapidaai/react';
import { Card } from '@/app/components/base/cards';
import { KnowledgeDropdown } from '@/app/components/Dropdown/knowledge-dropdown';
import { FormLabel } from '@/app/components/form-label';
import CheckboxCard from '@/app/components/Form/checkbox-card';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Slider } from '@/app/components/Form/Slider';
import { HybridSearchIcon } from '@/app/components/Icon/hybrid-search';
import { TextSearchIcon } from '@/app/components/Icon/text-search';
import { VectorSearchIcon } from '@/app/components/Icon/vector-search';
import { Tooltip } from '@/app/components/Tooltip';
import { RETRIEVE_METHOD } from '@/models/datasets';
import { cn } from '@/styles/media';
import { ExternalLink, Info, InfoIcon } from 'lucide-react';
import { InputGroup } from '@/app/components/input-group';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { Textarea } from '@/app/components/Form/Textarea';
import { CodeEditor } from '@/app/components/Form/editor/code-editor';

export const ConfigureKnowledgeRetrieval: React.FC<{
  inputClass?: string;
  toolDefinition: {
    name: string;
    description: string;
    parameters: string;
  };
  onChangeToolDefinition: (vl: {
    name: string;
    description: string;
    parameters: string;
  }) => void;
  onParameterChange: (parameters: Metadata[]) => void;
  parameters: Metadata[] | null;
}> = ({
  toolDefinition,
  onChangeToolDefinition,
  inputClass,
  onParameterChange,
  parameters,
}) => {
  const getParamValue = (key: string) => {
    return parameters?.find(p => p.getKey() === key)?.getValue() ?? '';
  };

  //
  const updateParameter = (key: string, value: string) => {
    const updatedParams = [...(parameters || [])];
    const existingIndex = updatedParams.findIndex(p => p.getKey() === key);
    const newParam = new Metadata();
    newParam.setKey(key);
    newParam.setValue(value);
    if (existingIndex >= 0) {
      updatedParams[existingIndex] = newParam;
    } else {
      updatedParams.push(newParam);
    }
    onParameterChange(updatedParams);
  };
  return (
    <>
      <InputGroup title="Action Definition">
        <div className={cn('p-6 pt-2 flex flex-col gap-8 max-w-6xl')}>
          <KnowledgeDropdown
            className={inputClass}
            currentKnowledge={getParamValue('tool.knowledge_id')}
            onChangeKnowledge={(kn: Knowledge) => {
              if (kn) updateParameter('tool.knowledge_id', kn.getId());
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
                wrapperClassNames={'h-auto'}
                checked={
                  getParamValue('tool.search_type') === RETRIEVE_METHOD.hybrid
                }
                onChange={() => {
                  updateParameter('tool.search_type', RETRIEVE_METHOD.hybrid);
                }}
              >
                <Card
                  className={cn(
                    'p-3 flex flex-row space-x-3 bg-light-background',
                    'h-auto',
                    inputClass,
                  )}
                >
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
                wrapperClassNames={cn('h-auto')}
                checked={
                  getParamValue('tool.search_type') === RETRIEVE_METHOD.semantic
                }
                onChange={() => {
                  updateParameter('tool.search_type', RETRIEVE_METHOD.semantic);
                }}
              >
                <Card
                  className={cn(
                    'p-3 flex flex-row space-x-3 bg-light-background',
                    inputClass,
                  )}
                >
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
                wrapperClassNames={'h-auto'}
                value="text"
                checked={
                  getParamValue('tool.search_type') === RETRIEVE_METHOD.fullText
                }
                onChange={() => {
                  updateParameter('tool.search_type', RETRIEVE_METHOD.fullText);
                }}
              >
                <Card
                  className={cn(
                    'p-3 flex flex-row space-x-3 bg-light-background',
                    inputClass,
                  )}
                >
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
                  value={getParamValue('tool.top_k')}
                  onSlide={(c: number) => {
                    updateParameter('tool.top_k', c.toString());
                  }}
                />
                <Input
                  id="top_k"
                  className={cn(
                    'py-0 px-1 tabular-nums w-10 h-6 text-xs bg-light-background',
                    inputClass,
                  )}
                  min={0}
                  max={10}
                  type="number"
                  value={Number(getParamValue('tool.top_k'))}
                  onChange={c => {
                    updateParameter('tool.top_k', c.target.value);
                  }}
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
                  onSlide={(c: number) => {
                    updateParameter('tool.score_threshold', c.toString());
                  }}
                  value={getParamValue('tool.score_threshold')}
                />
                <Input
                  id="score_threshold"
                  className={cn(
                    'py-0 px-1 tabular-nums w-10 h-6 text-xs bg-light-background',
                    inputClass,
                  )}
                  min={0}
                  max={1}
                  type="number"
                  step=".01"
                  value={getParamValue('tool.score_threshold')}
                  onChange={c => {
                    updateParameter('tool.score_threshold', c.target.value);
                  }}
                />
              </div>
            </FieldSet>
          </div>
        </div>
      </InputGroup>
      <InputGroup title="Tool Definition">
        <YellowNoticeBlock className="flex items-center">
          <Info className="shrink-0 w-4 h-4" />
          <div className="ms-3 text-sm font-medium">
            Know more about knowledge tool definiation that can be supported by
            rapida
          </div>
          <a
            target="_blank"
            href="https://doc.rapida.ai/assistants/overview"
            className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
            rel="noreferrer"
          >
            Read documentation
            <ExternalLink
              className="shrink-0 w-4 h-4 ml-1.5"
              strokeWidth={1.5}
            />
          </a>
        </YellowNoticeBlock>
        <div className={cn('p-6 flex flex-col gap-8 max-w-6xl')}>
          <FieldSet className="relative w-full">
            <FormLabel>Name</FormLabel>
            <Input
              value={toolDefinition.name}
              onChange={e =>
                onChangeToolDefinition({
                  ...toolDefinition,
                  name: e.target.value,
                })
              }
              placeholder="Enter tool name"
              className={cn('bg-light-background', inputClass)}
            />
          </FieldSet>
          <FieldSet className="relative w-full">
            <FormLabel>Description</FormLabel>
            <Textarea
              value={toolDefinition.description}
              onChange={e =>
                onChangeToolDefinition({
                  ...toolDefinition,
                  description: e.target.value,
                })
              }
              className={cn('bg-light-background', inputClass)}
              placeholder="A tool description or definition of when this tool will get triggered."
              rows={2}
            />
          </FieldSet>

          <FieldSet className="relative w-full">
            <FormLabel>Parameters</FormLabel>
            <CodeEditor
              placeholder="Provide a tool parameters that will be passed to llm"
              value={toolDefinition.parameters}
              onChange={value => {
                onChangeToolDefinition({
                  ...toolDefinition,
                  parameters: value,
                });
              }}
              className={cn(
                'min-h-40 max-h-dvh bg-light-background dark:bg-gray-950',
                inputClass,
              )}
            />
          </FieldSet>
        </div>
      </InputGroup>
    </>
  );
};

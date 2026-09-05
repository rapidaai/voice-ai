import { FC } from 'react';
import { Knowledge } from '@rapidaai/react';
import { Information, Checkmark } from '@carbon/icons-react';
import { cn } from '@/utils';
import { CornerBorderOverlay } from '@/app/components/base/corner-border';
import { KnowledgeDropdown } from '@/app/components/dropdown/knowledge-dropdown';
import { Slider } from '@carbon/react';
import { HybridSearchIcon } from '@/app/components/Icon/hybrid-search';
import { TextSearchIcon } from '@/app/components/Icon/text-search';
import { VectorSearchIcon } from '@/app/components/Icon/vector-search';
import { Tooltip } from '@carbon/react';
import { RETRIEVE_METHOD } from '@/models/datasets';
import {
  ConfigureToolProps,
  ToolDefinitionForm,
  useParameterManager,
} from '../common';
import { Stack } from '../../carbon/form';
import { InputGroup } from '../../input-group/index';

// ============================================================================
// Constants
// ============================================================================

const SEARCH_TYPE_CONFIG = [
  {
    id: 'hybrid-search-type',
    value: RETRIEVE_METHOD.hybrid,
    icon: HybridSearchIcon,
    title: 'Hybrid Search',
    description:
      "Execute full-text search and vector searches simultaneously, re-rank to select the best match for the user's query.",
  },
  {
    id: 'vector-search-type',
    value: RETRIEVE_METHOD.semantic,
    icon: VectorSearchIcon,
    title: 'Semantic Search',
    description:
      'Generate query embeddings and search for the text chunk most similar to its vector representation.',
  },
  {
    id: 'text-search-type',
    value: RETRIEVE_METHOD.fullText,
    icon: TextSearchIcon,
    title: 'Full Text Search',
    description:
      'Index all terms in the document, allowing users to search any term and retrieve relevant text chunk containing those terms.',
  },
] as const;

// ============================================================================
// Main Component
// ============================================================================

export const ConfigureKnowledgeRetrieval: FC<ConfigureToolProps> = ({
  toolDefinition,
  onChangeToolDefinition,
  inputClass,
  onParameterChange,
  parameters,
}) => {
  const { getParamValue, updateParameter } = useParameterManager(
    parameters,
    onParameterChange,
  );

  return (
    <>
      <InputGroup title="Action Definition">
        <Stack gap={7}>
          <KnowledgeDropdown
            className={cn('bg-light-background', inputClass)}
            currentKnowledge={getParamValue('tool.knowledge_id')}
            onChangeKnowledge={(knowledge: Knowledge) => {
              if (knowledge) {
                updateParameter('tool.knowledge_id', knowledge.getId());
              }
            }}
          />

          <div>
            <p className="text-xs font-medium mb-2">Retrieval setting</p>
            <div className="grid grid-cols-3 gap-3">
              {SEARCH_TYPE_CONFIG.map(config => (
                <SearchTypeCard
                  key={config.id}
                  {...config}
                  isSelected={
                    getParamValue('tool.search_type') === config.value
                  }
                  onSelect={() =>
                    updateParameter('tool.search_type', config.value)
                  }
                />
              ))}
            </div>
          </div>

          <div className="grid grid-cols-2 w-full gap-4">
            <SliderField
              id="top-k"
              label="Top K"
              tooltip="Used to filter chunks that are most similar to user questions. The system will also dynamically adjust the value of Top K, according to max_tokens of the selected model."
              min={1}
              max={10}
              step={1}
              value={getParamValue('tool.top_k')}
              onChange={value => updateParameter('tool.top_k', value)}
            />

            <SliderField
              id="score-threshold"
              label="Score Threshold"
              tooltip="Used to filter chunks that are most similar to user questions. The system will also dynamically adjust the value of Top K, according to max_tokens of the selected model."
              min={0}
              max={1}
              step={0.01}
              value={getParamValue('tool.score_threshold')}
              onChange={value => updateParameter('tool.score_threshold', value)}
            />
          </div>
        </Stack>
      </InputGroup>

      {toolDefinition && onChangeToolDefinition && (
        <ToolDefinitionForm
          toolDefinition={toolDefinition}
          onChangeToolDefinition={onChangeToolDefinition}
          inputClass={inputClass}
          documentationPath="/assistants/tools/add-knowledge-tool"
        />
      )}
    </>
  );
};

// ============================================================================
// Search Type Card
// ============================================================================

interface SearchTypeCardProps {
  id: string;
  value: string;
  icon: FC<{ className?: string }>;
  title: string;
  description: string;
  isSelected: boolean;
  onSelect: () => void;
}

const SearchTypeCard: FC<SearchTypeCardProps> = ({
  id,
  icon: Icon,
  title,
  description,
  isSelected,
  onSelect,
}) => (
  <button
    type="button"
    onClick={onSelect}
    className={cn(
      'relative group text-left p-4 border transition-colors duration-100',
      isSelected
        ? 'border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950/50'
        : 'border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-950/50 hover:bg-gray-50 dark:hover:bg-gray-900/60',
    )}
  >
    <CornerBorderOverlay className={isSelected ? 'opacity-100' : undefined} />
    {isSelected && (
      <span className="absolute top-3 right-3 h-5 w-5 inline-flex items-center justify-center bg-primary z-20">
        <Checkmark size={14} className="text-white" />
      </span>
    )}
    <div className="flex items-center gap-3 mb-2">
      <div className="flex items-center justify-center bg-blue-100 dark:bg-blue-900/30 shrink-0 h-8 w-8">
        <Icon className="text-blue-600" />
      </div>
      <span className="text-sm font-medium">{title}</span>
    </div>
    <p className="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">
      {description}
    </p>
  </button>
);

// ============================================================================
// Slider Field
// ============================================================================

interface SliderFieldProps {
  id: string;
  label: string;
  tooltip: string;
  min: number;
  max: number;
  step: number;
  value: string;
  onChange: (value: string) => void;
}

const SliderField: FC<SliderFieldProps> = ({
  id,
  label,
  tooltip,
  min,
  max,
  step,
  value,
  onChange,
}) => (
  <div className="[&_.cds--slider-container]:!mt-0 [&_.cds--slider__range-label]:hidden">
    <Slider
      id={id}
      labelText={
        <Tooltip align="top" label={tooltip}>
          <span className="inline-flex items-center gap-1">
            {label}
            <Information size={14} />
          </span>
        </Tooltip>
      }
      min={min}
      max={max}
      step={step}
      value={Number(value) || min}
      onChange={({ value: v }: { value: number }) => onChange(v.toString())}
    />
  </div>
);

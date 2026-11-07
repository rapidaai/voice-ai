import type { ComponentType } from 'react';
import { Knowledge } from '@rapidaai/react';
import {
  DataBase,
  Information,
  Checkmark,
  ModelAlt,
  Search,
} from '@carbon/icons-react';
import { cn } from '@/utils';
import { CornerBorderOverlay } from '@/app/components/ui/primitives';
import { KnowledgeDropdown } from '@/app/components/domain/dropdowns/knowledge-dropdown';
import { SelectableTile, Slider } from '@carbon/react';
import { Tooltip } from '@carbon/react';
import { RETRIEVE_METHOD } from '@/models/datasets';
import {
  ConfigureToolProps,
  ToolDefinitionForm,
  useParameterManager,
} from './common';
import { Stack } from '@/app/components/ui/primitives';
import { InputGroup } from '@/app/components/ui/primitives';

// ============================================================================
// Constants
// ============================================================================

const SEARCH_TYPE_CONFIG = [
  {
    id: 'hybrid-search-type',
    value: RETRIEVE_METHOD.hybrid,
    icon: DataBase,
    title: 'Hybrid Search',
    description:
      "Execute full-text search and vector searches simultaneously, re-rank to select the best match for the user's query.",
  },
  {
    id: 'vector-search-type',
    value: RETRIEVE_METHOD.semantic,
    icon: ModelAlt,
    title: 'Semantic Search',
    description:
      'Generate query embeddings and search for the text chunk most similar to its vector representation.',
  },
  {
    id: 'text-search-type',
    value: RETRIEVE_METHOD.fullText,
    icon: Search,
    title: 'Full Text Search',
    description:
      'Index all terms in the document, allowing users to search any term and retrieve relevant text chunk containing those terms.',
  },
] as const;

// ============================================================================
// Main Component
// ============================================================================

export function ConfigureKnowledgeRetrieval({
  toolDefinition,
  onChangeToolDefinition,
  inputClass,
  onParameterChange,
  parameters,
}: ConfigureToolProps) {
  const { getParamValue, updateParameter } = useParameterManager(
    parameters,
    onParameterChange,
  );

  return (
    <>
      <InputGroup title="Action Definition">
        <Stack gap={7}>
          <KnowledgeDropdown
            className={inputClass}
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
}

// ============================================================================
// Search Type Card
// ============================================================================

interface SearchTypeCardProps {
  id: string;
  value: string;
  icon: ComponentType<{ className?: string }>;
  title: string;
  description: string;
  isSelected: boolean;
  onSelect: () => void;
}

function SearchTypeCard({
  id,
  value,
  icon: Icon,
  title,
  description,
  isSelected,
  onSelect,
}: SearchTypeCardProps) {
  return (
    <SelectableTile
      id={id}
      data-value={value}
      selected={isSelected}
      onClick={onSelect}
      className={cn(
        'relative group !min-h-0 !p-4 !text-left !transition-colors !duration-100',
        isSelected
          ? '!border-border-subtle !bg-surface '
          : '!border-border-subtle !bg-surface hover:!bg-layer ',
      )}
    >
      <CornerBorderOverlay className={isSelected ? 'opacity-100' : undefined} />
      {isSelected && (
        <span className="absolute top-3 right-3 h-5 w-5 inline-flex items-center justify-center bg-primary z-20">
          <Checkmark size={14} className="text-inverse" />
        </span>
      )}
      <div className="flex items-center gap-3 mb-2">
        <div className="flex items-center justify-center bg-support-info/10 shrink-0 h-8 w-8">
          <Icon className="text-support-info" />
        </div>
        <span className="text-sm font-medium">{title}</span>
      </div>
      <p className="text-xs text-muted leading-relaxed">{description}</p>
    </SelectableTile>
  );
}

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

function SliderField({
  id,
  label,
  tooltip,
  min,
  max,
  step,
  value,
  onChange,
}: SliderFieldProps) {
  return (
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
}

import React, { useRef, useState, useEffect } from 'react';
import { Info, Search, VolumeX, X } from 'lucide-react';
import {
  ConnectionConfig,
  Criteria,
  GetAllAssistantTelemetry,
  GetAllAssistantTelemetryRequest,
  Paginate,
} from '@rapidaai/react';
import { ModalProps } from '@/app/components/base/modal';
import { connectionConfig } from '@/configs';
import { BottomModal } from '@/app/components/base/modal/bottom-side-modal';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { PaginationButtonBlock } from '@/app/components/blocks/pagination-button-block';
import { IButton } from '@/app/components/Form/Button';
import { ChevronRight } from 'lucide-react';
import { useCurrentCredential } from '@/hooks/use-credential';
import { formatNanoToReadableMilli } from '@/utils';
import {
  cn,
  formatDateWithMillisecond,
  toDate,
  toDateWithMicroseconds,
} from '@/styles/media';
import { Tooltip } from '@/app/components/base/tooltip';
import { CodeHighlighting } from '@/app/components/code-highlighting';
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb';

interface ConversationTelemetryDialogProps extends ModalProps {
  criterias?: Criteria[]; // Added criteria prop to accept Criteria from parent
}

interface Chip {
  field: string;
  value: string | number;
  id: string;
}

interface SearchField {
  id: string;
  label: string;
  type: 'string' | 'number' | 'select';
  placeholder?: string;
  options?: string[];
}

interface SentrySearchProps {
  className?: string;
  updateChips: (chips: Chip[]) => void; // Callback to update chips in parent
  existingChips: Chip[];
}

interface SpanData {
  stageName: string;
  spanId: string;
  parentId: string | null;
  start: Timestamp;
  end: Timestamp;
  duration: string;
  attributes: string;
  children: any[];
}

export function ConversationTelemetryDialog(
  props: ConversationTelemetryDialogProps,
) {
  const { token, authId, projectId } = useCurrentCredential();
  const [chips, setChips] = useState<Chip[]>([]); // State for chips
  const [timeline, setTimeline] = useState<SpanData[]>([]);
  const [expandedSpans, setExpandedSpans] = useState<Set<string>>(new Set());

  const toggleExpand = (spanId: string) => {
    setExpandedSpans(prev => {
      const newExpandedSpans = new Set(prev);
      if (newExpandedSpans.has(spanId)) {
        newExpandedSpans.delete(spanId);
      } else {
        newExpandedSpans.add(spanId);
      }
      return newExpandedSpans;
    });
  };

  useEffect(() => {
    const initialChips = (props.criterias || []).map((criteria, index) => ({
      field: criteria.getKey(),
      value: criteria.getValue(),
      id: `${Date.now()}-${index}`, // Ensure a unique ID using timestamp and index
    }));
    setChips(initialChips);
  }, [props.criterias]);

  const updateChips = (newChips: Chip[]) => {
    setChips(newChips);
  };

  useEffect(() => {
    const request = new GetAllAssistantTelemetryRequest();
    const paginate = new Paginate();
    paginate.setPage(1);
    paginate.setPagesize(50);
    request.setPaginate(paginate);

    const criteriaList = [
      ...chips.map(chip => {
        const criteria = new Criteria();
        criteria.setKey(chip.field); // Assuming chip.field maps correctly
        criteria.setValue(String(chip.value));
        criteria.setLogic('match'); // Default logic
        return criteria;
      }),
    ];

    request.setCriteriasList(criteriaList);

    GetAllAssistantTelemetry(
      connectionConfig,
      request,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: authId,
        projectId: projectId,
      }),
    ).then(response => {
      console.dir(response.toObject());
      const telemetryList = response.getDataList() || [];
      const spanMap: Record<string, any[]> = { root: [] };

      telemetryList.forEach(telemetry => {
        const startTime = telemetry.getStarttime();
        const endTime = telemetry.getEndtime();

        if (!startTime || !endTime) {
          console.warn('Telemetry missing start or end time:', telemetry);
          return;
        }
        const spanData: SpanData = {
          stageName: telemetry.getStagename(),
          spanId: telemetry.getSpanid(),
          parentId: telemetry.getParentid() || 'root', // Default to "root" if parentId is missing
          start: telemetry.getStarttime()!,
          end: telemetry.getEndtime()!,
          duration: formatNanoToReadableMilli(telemetry.getDuration(), 3),
          attributes: JSON.stringify(
            telemetry.getAttributesMap().toObject(),
            undefined,
            2,
          ),
          children: [],
        };

        const parentId = telemetry.getParentid() || 'root';
        if (!spanMap[parentId]) {
          spanMap[parentId] = []; // Create an array for orphans
        }
        spanMap[parentId].push(spanData);
      });

      // Add orphan spans to root if their parent IDs are completely missing
      const allSpanIds = new Set(telemetryList.map(t => t.getSpanid()));
      Object.keys(spanMap).forEach(parentId => {
        if (parentId !== 'root' && !allSpanIds.has(parentId)) {
          spanMap['root'] = [...(spanMap['root'] || []), ...spanMap[parentId]];
          delete spanMap[parentId]; // Cleanup orphan group
        }
      });

      const buildHierarchy = (id: string): SpanData[] => {
        const spanList = spanMap[id] || [];

        // Sort spans by their start timestamp
        spanList.sort((a, b) => {
          const aStart = a.start.toDate();
          const bStart = b.start.toDate();
          return aStart.getTime() - bStart.getTime();
        });

        return spanList.map(span => ({
          ...span,
          children: buildHierarchy(span.spanId),
        }));
      };

      setTimeline(buildHierarchy('root'));
    });
  }, [token, authId, projectId, JSON.stringify(chips)]);

  const getTimelinePosition = (stage: SpanData) => {
    // Helper function to flatten spans and children into a flat array
    const flattenSpans = (spans: SpanData[]): SpanData[] =>
      spans.reduce(
        (acc: SpanData[], span: SpanData) => [
          ...acc,
          span,
          ...flattenSpans(span.children),
        ],
        [],
      );

    // Flatten all spans recursively from the timeline
    const allSpans = flattenSpans(timeline);

    // Calculate minimum and maximum times from flattened spans
    const minTime = Math.min(...allSpans.map(s => s.start.toDate().getTime()));
    const maxTime = Math.max(...allSpans.map(s => s.end.toDate().getTime()));
    const totalDuration = maxTime - minTime;

    // Extract the start and end times for the current stage
    const startMs = stage.start.toDate().getTime();
    const endMs = stage.end.toDate().getTime();

    // Calculate relative position within the timeline
    const left = ((startMs - minTime) / totalDuration) * 100;
    const width = ((endMs - startMs) / totalDuration) * 100;

    return { left, width };
  };

  const renderTimeline = (spans: SpanData[], level = 0) => {
    return spans.map(span => {
      const isExpanded = expandedSpans.has(span.spanId);
      const hasChildren = span.children.length > 0;
      const timelinePos = getTimelinePosition(span);

      return (
        <div key={span.spanId}>
          <div
            className={cn(
              'grid grid-cols-9 gap-4 p-4 cursor-pointer transition-colors dark:hover:bg-gray-950/40 hover:bg-gray-100',
            )}
            style={{ paddingLeft: `${level * 20 + 6}px` }}
            onClick={() => toggleExpand(span.spanId)}
          >
            <div className="flex items-center gap-4 col-span-1">
              <div
                className={cn(
                  'flex items-center  underline',
                  hasChildren ? 'text-blue-600' : 'opacity-0',
                )}
              >
                <span className="h-6 w-6 flex items-center justify-center rounded-[2px] hover:bg-gray-300 dark:hover:bg-gray-800">
                  <ChevronRight
                    strokeWidth={1.5}
                    className={cn(
                      'h-full w-full transition-all',
                      isExpanded && 'rotate-90',
                    )}
                  />
                </span>
              </div>

              <span className="truncate font-medium text-blue-600 ">
                {span.spanId.split('-')[0]}
              </span>
            </div>

            <div className="flex items-center col-span-2 truncate space-x-2">
              <span>generic</span> / {span.stageName}
              <Tooltip
                className="p-0"
                content={
                  <CodeHighlighting
                    language="json"
                    className="w-[700px] h-64 p-4 m-0"
                    code={span.attributes}
                  ></CodeHighlighting>
                }
              >
                <IButton type="button" className="h-6 w-6 p-0">
                  <Info className="w-4 h-4 opacity-60" strokeWidth={1.5} />
                </IButton>
              </Tooltip>
            </div>
            <div className="flex items-center col-span-3">
              <div className="relative h-6 bg-blue-600/10 w-full">
                <div
                  className="absolute h-full bg-blue-600 opacity-80 hover:opacity-100 transition-opacity"
                  style={{
                    left: `${timelinePos.left}%`,
                    width: `${timelinePos.width}%`,
                    minWidth: '2px',
                  }}
                />
              </div>
            </div>
            <div className="flex items-center col-span-1">
              <span className="font-semibold text-sm">{span.duration}</span>
            </div>
            <div className="flex items-center col-span-1">
              <span className="text-sm">
                {formatDateWithMillisecond(toDate(span.start))}
              </span>
            </div>
            <div className="flex items-center col-span-1">
              <span className="text-sm">
                {formatDateWithMillisecond(toDate(span.end))}
              </span>
            </div>
          </div>

          {isExpanded && (
            <div className="bg-gray-100 dark:bg-gray-950">
              {renderTimeline(span.children, level + 1)}
            </div>
          )}
        </div>
      );
    });
  };

  return (
    <BottomModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-full flex-1 h-[75vh]"
    >
      <ModalBody className="px-0 flex-1 space-y-0">
        <div className="sticky top-2 z-10">
          <BluredWrapper className="border-t">
            <div className="flex flex-1">
              <SentrySearch
                className="bg-light-background"
                updateChips={updateChips}
                existingChips={chips}
              />
            </div>
            <PaginationButtonBlock>
              <IButton
                onClick={() => {
                  props.setModalOpen(false);
                }}
              >
                <X strokeWidth={1.5} className="h-4 w-4" />
              </IButton>
            </PaginationButtonBlock>
          </BluredWrapper>
        </div>
        <div className="divide-y divide-gray-100 dark:divide-gray-800">
          {renderTimeline(timeline)}
        </div>
      </ModalBody>
    </BottomModal>
  );
}

export function SentrySearch({
  className,
  updateChips,
  existingChips,
}: SentrySearchProps) {
  const [showFieldDropdown, setShowFieldDropdown] = useState<boolean>(false);
  const [showValueInput, setShowValueInput] = useState<boolean>(false);
  const [selectedField, setSelectedField] = useState<SearchField | null>(null);
  const [inputValue, setInputValue] = useState<string>('');
  const [chips, setChips] = useState<Chip[]>([]);
  const [showStageDropdown, setShowStageDropdown] = useState<boolean>(false);

  const dropdownRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);
  const stageDropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setChips(existingChips);
  }, [existingChips]);
  const searchFields: SearchField[] = [
    { id: 'id', label: 'ID', type: 'string', placeholder: 'Enter UUID...' },
    {
      id: 'stageName',
      label: 'StageName',
      type: 'select',
      options: [
        'talk.assistant.connect',
        'talk.assistant.connect.create-conversation',
        'talk.assistant.connect.resume-conversation',
        'talk.assistant.listen.connect',
        'talk.assistant.speak.connect',
        'talk.assistant.listen.listening',
        'talk.assistant.utterance',
        'talk.assistant.interrupt',
        'talk.assistant.agent.connect',
        'talk.assistant.tool.connect',
        'talk.assistant.tool.execute',
        'talk.assistant.agent.text-generation',
        'talk.assistant.speak.transcribe',
        'talk.assistant.speak.speaking',
        'talk.assistant.notify',
        'talk.assistant.disconnect',
      ],
    },
    {
      id: 'assistantId',
      label: 'AssistantId',
      type: 'number',
      placeholder: 'Enter ID...',
    },
    {
      id: 'assistantProviderModelId',
      label: 'AssistantProviderModelId',
      type: 'number',
      placeholder: 'Enter ID...',
    },
    {
      id: 'assistantConversationId',
      label: 'AssistantConversationId',
      type: 'number',
      placeholder: 'Enter ID...',
    },
    {
      id: 'attributes.messageId',
      label: 'MessageID',
      type: 'string',
      placeholder: 'Enter UUID...',
    },
  ];

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setShowFieldDropdown(false);
      }
      if (
        stageDropdownRef.current &&
        !stageDropdownRef.current.contains(event.target as Node)
      ) {
        setShowStageDropdown(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  useEffect(() => {
    if (showValueInput && inputRef.current) {
      inputRef.current.focus();
    }
  }, [showValueInput]);

  const handleFieldSelect = (field: SearchField) => {
    setSelectedField(field);
    setShowFieldDropdown(false);
    if (field.type === 'select') {
      setShowStageDropdown(true);
    } else {
      setShowValueInput(true);
    }
    setInputValue('');
  };

  const handleValueSubmit = (value: string = inputValue) => {
    if (selectedField && value.trim()) {
      const newChip = {
        field: selectedField.id,
        value: value.trim(),
        id: `${Date.now()}`,
      };
      const newChips = [...chips, newChip];
      setChips(newChips);
      updateChips(newChips); // Notify parent with updated chips
      setSelectedField(null);
      setShowValueInput(false);
      setInputValue('');
    }
  };

  const handleStageSelect = (stage: string) => {
    handleValueSubmit(stage);
    setShowStageDropdown(false);
  };

  const handleKeyPress = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleValueSubmit();
    } else if (e.key === 'Escape') {
      setShowValueInput(false);
      setSelectedField(null);
      setInputValue('');
    }
  };

  const removeChip = (chipId: string) => {
    const filteredChips = chips.filter(chip => chip.id !== chipId);
    setChips(filteredChips);
    updateChips(filteredChips); // Notify parent to delete chip
  };

  return (
    <div className="relative w-full flex-1">
      <div
        onClick={() => !showValueInput && setShowFieldDropdown(true)}
        className={cn(
          'form-input',
          'min-h-10',
          'dark:placeholder-gray-600 placeholder-gray-400',
          'dark:text-gray-300 text-gray-600',

          'flex items-center',
          'outline-solid outline-[1.5px] outline-transparent',
          'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
          'border-b border-gray-400 dark:border-gray-600',
          'dark:focus:border-blue-600 focus:border-blue-600',
          'transition-all duration-200 ease-in-out',

          'bg-light-background dark:bg-gray-950',
        )}
      >
        <Search className="w-4 h-4 text-gray-400 ml-3" />
        <div className="flex-1 flex flex-wrap items-center gap-2 py-2 px-2">
          {chips.map(chip => (
            <div
              key={chip.id}
              className="inline-flex items-center gap-2 px-2.5 py-1 rounded-[2px] text-sm border dark:border-gray-900 bg-blue-600/10 "
              onClick={e => e.stopPropagation()}
            >
              <span className="font-medium opacity-90 text-blue-600">
                {chip.field}:
              </span>
              <span className="text-blue-600">{chip.value}</span>
              <button
                onClick={() => removeChip(chip.id)}
                className="hover:bg-red-600 rounded-[2px] p-0.5 hover:text-white cursor-pointer"
              >
                <X className="w-3 h-3" strokeWidth={1.5} />
              </button>
            </div>
          ))}

          {showValueInput && selectedField && (
            <div
              className="inline-flex items-center gap-2 px-2.5 py-1 rounded-[2px] text-sm border "
              onClick={e => e.stopPropagation()}
            >
              <span className="font-medium opacity-90">
                {selectedField.label}:
              </span>
              <input
                ref={inputRef}
                type={selectedField.type === 'number' ? 'number' : 'text'}
                value={inputValue}
                onChange={e => setInputValue(e.target.value)}
                onKeyDown={handleKeyPress}
                onBlur={() => {
                  if (inputValue.trim()) {
                    handleValueSubmit();
                  } else {
                    setShowValueInput(false);
                    setSelectedField(null);
                  }
                }}
                placeholder={selectedField.placeholder}
                className="bg-transparent outline-hidden w-48"
              />
            </div>
          )}

          {!showValueInput && chips.length === 0 && (
            <span className="text-sm opacity-60">Click to add filters...</span>
          )}
        </div>
      </div>

      {showFieldDropdown && (
        <div
          ref={dropdownRef}
          className="absolute left-0 right-0 mt-2 bg-white dark:bg-gray-950 border divide-y dark:divide-gray-900 rounded-[2px] shadow-lg z-999"
        >
          <div className="px-3 py-2 bg-gray-100 dark:bg-gray-900">
            <span className="text-xs font-medium text-gray-500">
              SELECT FIELD
            </span>
          </div>
          {searchFields.map(field => (
            <button
              key={field.id}
              onClick={() => handleFieldSelect(field)}
              className="w-full text-left px-4 py-2.5 text-sm transition-colors flex items-center justify-between group"
            >
              <span className="font-medium  group-hover:text-blue-700">
                {field.label}
              </span>
              <span className="text-xs  group-hover:text-blue-500">
                {field.type === 'select' ? 'select' : field.type}
              </span>
            </button>
          ))}
        </div>
      )}

      {showStageDropdown && selectedField?.type === 'select' && (
        <div
          ref={stageDropdownRef}
          className="absolute left-0 right-0 mt-2 bg-white dark:bg-gray-950 border divide-y dark:divide-gray-900 rounded-[2px] shadow-lg z-999"
        >
          <div className="px-3 py-2 bg-gray-100 dark:bg-gray-900">
            <span className="text-xs font-medium">SELECT STAGE</span>
          </div>
          {(selectedField.options || []).map(option => (
            <button
              key={option}
              onClick={() => handleStageSelect(option)}
              className="w-full text-left px-4 py-2.5 text-sm hover:text-blue-700 transition-colors"
            >
              {option}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

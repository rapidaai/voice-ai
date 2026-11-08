import type { HTMLAttributes } from 'react';
import {
  BaseCard,
  CardDescription,
  CardTitle,
} from '@/app/components/ui/primitives';
import { cn } from '@/utils';
import type { AssistantTool } from '@rapidaai/react';
import { BUILDIN_TOOLS } from '@/llm-tools';
import { Tag, ButtonSet } from '@carbon/react';
import {
  getToolConditionSource,
  getToolConditionSourceLabel,
} from '@/app/components/domain/tools/common';
import {
  PrimaryButton,
  DangerGhostButton,
} from '@/app/components/ui/primitives';
import { Edit, TrashCan } from '@carbon/icons-react';
import type { Metadata } from '@rapidaai/react';

type PlainToolCardData = {
  name?: string;
  description?: string;
  buildinToolConfig?: {
    parameters?: Metadata[] | null;
  };
};

type ToolCardData = AssistantTool | PlainToolCardData;

interface ToolCardProps extends HTMLAttributes<HTMLDivElement> {
  tool: ToolCardData;
  onEdit?: () => void;
  onDelete?: () => void;
  iconClass?: string;
  titleClass?: string;
  isConnected?: boolean;
}

const hasAssistantToolMethods = (tool: ToolCardData): tool is AssistantTool =>
  typeof (tool as AssistantTool).getExecutionmethod === 'function';

export function SelectToolCard({
  tool,
  onEdit,
  onDelete,
  className,
}: ToolCardProps) {
  const hasProtobufMethods = hasAssistantToolMethods(tool);
  const executionMethod = hasProtobufMethods ? tool.getExecutionmethod() : '';
  const isMCP = executionMethod === 'mcp';

  const toolName = hasProtobufMethods ? tool.getName?.() : tool.name;
  const toolDescription = hasProtobufMethods
    ? tool.getDescription?.()
    : tool.description;
  const conditionSource = hasProtobufMethods
    ? getToolConditionSource(tool.getExecutionoptionsList?.())
    : getToolConditionSource(tool.buildinToolConfig?.parameters || []);

  const toolMeta = BUILDIN_TOOLS.find(x => x.code === executionMethod);

  return (
    <BaseCard className={cn('flex flex-col', className)}>
      <div className="p-4 flex-1 flex flex-col gap-3">
        <header className="flex items-start justify-between">
          <div className="w-9 h-9 flex items-center justify-center bg-layer shrink-0">
            {toolMeta?.icon ? (
              <img
                alt={toolMeta.name}
                src={toolMeta.icon}
                className="w-5 h-5 object-contain"
              />
            ) : (
              <span className="text-xs font-semibold text-muted uppercase">
                {(toolName ?? '?').charAt(0)}
              </span>
            )}
          </div>
          <div className="flex items-center gap-1">
            {toolMeta && (
              <Tag type="gray" size="sm">
                {toolMeta.name}
              </Tag>
            )}
            {isMCP && (
              <Tag type="purple" size="sm">
                MCP
              </Tag>
            )}
            {!toolMeta && !isMCP && (
              <Tag type="gray" size="sm" className="capitalize">
                {executionMethod.replace(/_/g, ' ')}
              </Tag>
            )}
            {conditionSource !== 'all' && (
              <Tag type="blue" size="sm">
                Source: {getToolConditionSourceLabel(conditionSource)}
              </Tag>
            )}
          </div>
        </header>

        <div className="flex-1 flex flex-col gap-1 min-w-0">
          <CardTitle className="line-clamp-1 text-sm font-semibold">
            {toolName}
          </CardTitle>
          <CardDescription className="line-clamp-2 text-xs leading-relaxed">
            {toolDescription}
          </CardDescription>
        </div>
      </div>

      <ButtonSet className="border-t border-border-subtle [&>button]:!flex-1 [&>button]:!max-w-none">
        {onDelete && (
          <DangerGhostButton size="md" renderIcon={TrashCan} onClick={onDelete}>
            Delete
          </DangerGhostButton>
        )}
        {onEdit && (
          <PrimaryButton size="md" renderIcon={Edit} onClick={onEdit}>
            Edit
          </PrimaryButton>
        )}
      </ButtonSet>
    </BaseCard>
  );
}

import React from 'react';
import {
  Button,
  StructuredListBody,
  StructuredListCell,
  StructuredListRow,
  StructuredListWrapper,
  Tag,
  Tile,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Add, Information } from '@carbon/icons-react';
import type { AgentTransition } from '@/app/pages/assistant/actions/agentflow';
import type { BaseNodeCardProps } from '../types';

type AgentNodeCardProps = BaseNodeCardProps & {
  agentModelLabel: string;
  agentPromptPreview: string;
  transitionRows: AgentTransition[];
  onAddTransition: (event: React.MouseEvent<HTMLButtonElement>) => void;
};

export function AgentNodeCard({
  toolbar,
  handles,
  Icon,
  cardClass,
  node,
  validationPill,
  width,
  agentModelLabel,
  agentPromptPreview,
  transitionRows,
  onAddTransition,
}: AgentNodeCardProps) {
  return (
    <>
      {toolbar}
      {handles}
      <Tile className={cardClass} style={{ width }}>
        <div className="flex h-12 items-center gap-3 border-b border-border-subtle px-4 text-foreground ">
          <Icon size={18} className="shrink-0" />
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <div className="min-w-0 truncate text-base font-semibold">
              {node.label || 'Agent'}
            </div>
            {validationPill}
          </div>
          <Tag
            size="sm"
            type="gray"
            title={agentModelLabel}
            className="!m-0 max-w-[112px] truncate"
          >
            {agentModelLabel}
          </Tag>
        </div>
        <div className="mx-4 mt-4">
          <div className="mb-2 flex items-center gap-1 text-xs font-semibold uppercase text-muted ">
            Instruction
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={12} />
              </ToggletipButton>
              <ToggletipContent className="normal-case">
                preview of the instruction used by this agent node.
              </ToggletipContent>
            </Toggletip>
          </div>
          <div className="min-h-[108px] border border-border-subtle bg-surface p-4 ">
            <div className="line-clamp-4 text-sm leading-5 text-foreground ">
              {agentPromptPreview || 'No instruction configured'}
            </div>
          </div>
        </div>
        <div className="flex h-12 items-center justify-between border-b border-border-subtle px-4 ">
          <div className="flex items-center gap-1 text-xs font-semibold uppercase text-muted ">
            Transitions
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={12} />
              </ToggletipButton>
              <ToggletipContent className="normal-case">
                output paths the agent can choose after this turn.
              </ToggletipContent>
            </Toggletip>
          </div>
          <Button
            kind="ghost"
            size="sm"
            hasIconOnly
            renderIcon={Add}
            iconDescription="Add transition"
            onClick={onAddTransition}
            className="nodrag !h-8 !min-h-8 !w-8 !p-0"
          />
        </div>
        <div>
          <StructuredListWrapper
            aria-label={`${node.label} transitions`}
            isCondensed
            isFlush
            className="!m-0 !w-full !min-w-0 !border-0"
          >
            <StructuredListBody>
              {transitionRows.map(transition => (
                <StructuredListRow
                  key={`${node.id}-transition-${transition.id}`}
                  className="!border-b-0"
                >
                  <StructuredListCell
                    noWrap
                    className="!flex !h-10 !items-center !gap-3 !px-7 !py-0 text-sm text-foreground "
                  >
                    <span className="text-base leading-none text-muted">=</span>
                    <span className="min-w-0 truncate">
                      {transition.name || 'Unnamed transition'}
                    </span>
                  </StructuredListCell>
                </StructuredListRow>
              ))}
            </StructuredListBody>
          </StructuredListWrapper>
        </div>
      </Tile>
    </>
  );
}

import {
  StructuredListBody,
  StructuredListCell,
  StructuredListRow,
  StructuredListWrapper,
  Tile,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';
import type { ChatInputArgument } from '@/app/pages/assistant/actions/agentflow';
import type { BaseNodeCardProps } from '../types';

type ChatInputNodeCardProps = BaseNodeCardProps & {
  inputArguments: ChatInputArgument[];
};

export function ChatInputNodeCard({
  toolbar,
  handles,
  Icon,
  cardClass,
  node,
  validationPill,
  width,
  inputArguments,
}: ChatInputNodeCardProps) {
  return (
    <>
      {toolbar}
      {handles}
      <Tile className={cardClass} style={{ width }}>
        <div className="flex h-12 items-center gap-3 border-b border-border-subtle px-4 text-foreground ">
          <Icon size={18} className="shrink-0" />
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <div className="min-w-0 truncate text-base font-semibold">
              {node.label || 'Chat Input'}
            </div>
            {validationPill}
          </div>
        </div>
        {inputArguments.length > 0 && (
          <>
            <div className="flex h-12 items-center justify-between border-b border-border-subtle px-4 ">
              <div className="flex items-center gap-1 text-xs font-semibold uppercase text-muted ">
                Arguments
                <Toggletip align="right">
                  <ToggletipButton label="Show information">
                    <Information size={12} />
                  </ToggletipButton>
                  <ToggletipContent className="normal-case">
                    values available when the workflow begins.
                  </ToggletipContent>
                </Toggletip>
              </div>
            </div>
            <div className="pb-4">
              <StructuredListWrapper
                aria-label={`${node.label} arguments`}
                isCondensed
                isFlush
                className="!m-0 !w-full !min-w-0 !border-0"
              >
                <StructuredListBody>
                  {inputArguments.map(argument => (
                    <StructuredListRow
                      key={`${node.id}-argument-${argument.id}`}
                      className="!border-b-0"
                    >
                      <StructuredListCell
                        noWrap
                        className="!flex !h-9 !items-center !justify-between !gap-3 !px-7 !py-0 text-sm text-foreground "
                      >
                        <span className="min-w-0 truncate">
                          {argument.name || 'Unnamed argument'}
                        </span>
                        <span className="shrink-0 text-xs text-muted ">
                          {argument.type || 'string'}
                        </span>
                      </StructuredListCell>
                    </StructuredListRow>
                  ))}
                </StructuredListBody>
              </StructuredListWrapper>
            </div>
          </>
        )}
      </Tile>
    </>
  );
}

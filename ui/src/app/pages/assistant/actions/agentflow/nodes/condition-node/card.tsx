import {
  Tile,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';
import type { BaseNodeCardProps } from '../types';

type ConditionNodeCardProps = BaseNodeCardProps & {
  outputRows: Array<{
    id: string;
    label: string;
    prefix: string;
  }>;
};

export function ConditionNodeCard({
  toolbar,
  handles,
  Icon,
  cardClass,
  node,
  validationPill,
  width,
  outputRows,
}: ConditionNodeCardProps) {
  return (
    <>
      {toolbar}
      {handles}
      <Tile className={cardClass} style={{ width }}>
        <div className="flex h-12 items-center gap-3 border-b border-border-subtle px-4 text-foreground ">
          <Icon size={18} className="shrink-0" />
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <div className="min-w-0 truncate text-base font-semibold">
              {node.label || 'If / Else'}
            </div>
            {validationPill}
          </div>
        </div>
        <div className="flex h-12 items-center justify-between border-b border-border-subtle px-4 ">
          <div className="flex items-center gap-1 text-xs font-semibold uppercase text-muted ">
            Condition
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={12} />
              </ToggletipButton>
              <ToggletipContent className="normal-case">
                outgoing paths evaluated by this condition.
              </ToggletipContent>
            </Toggletip>
          </div>
        </div>
        <div
          aria-label={`${node.label} conditions`}
          className="w-full min-w-0 overflow-hidden pb-4"
        >
          {outputRows.map(row => (
            <div
              key={`${node.id}-condition-row-${row.id}`}
              className="flex h-10 w-full min-w-0 items-center gap-3 overflow-hidden px-7 py-0 text-sm text-foreground "
            >
              <span className="w-8 shrink-0 text-xs font-semibold uppercase text-muted ">
                {row.prefix}
              </span>
              <span
                className="block min-w-0 flex-1 overflow-hidden truncate whitespace-nowrap"
                title={row.label}
              >
                {row.label}
              </span>
            </div>
          ))}
        </div>
      </Tile>
    </>
  );
}

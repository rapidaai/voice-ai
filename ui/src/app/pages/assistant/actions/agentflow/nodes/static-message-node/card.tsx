import {
  Tag,
  Tile,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { Information } from '@carbon/icons-react';
import type { BaseNodeCardProps } from '../types';

type StaticMessageNodeCardProps = BaseNodeCardProps & {
  messagePreview: string;
  postDelayMs: number;
};

export function StaticMessageNodeCard({
  toolbar,
  handles,
  Icon,
  cardClass,
  node,
  validationPill,
  width,
  messagePreview,
  postDelayMs,
}: StaticMessageNodeCardProps) {
  return (
    <>
      {toolbar}
      {handles}
      <Tile className={cardClass} style={{ width }}>
        <div className="flex h-12 items-center gap-3 border-b border-border-subtle px-4 text-foreground ">
          <Icon size={18} className="shrink-0" />
          <div className="flex min-w-0 flex-1 items-center gap-2">
            <div className="min-w-0 truncate text-base font-semibold">
              {node.label || 'Static Message'}
            </div>
            {validationPill}
          </div>
          {postDelayMs > 0 && (
            <Tag size="sm" type="gray" className="!m-0">
              {postDelayMs} ms
            </Tag>
          )}
        </div>
        <div className="mx-4 mt-4 pb-4">
          <div className="mb-2 flex items-center gap-1 text-xs font-semibold uppercase text-muted ">
            Message
            <Toggletip align="right">
              <ToggletipButton label="Show information">
                <Information size={12} />
              </ToggletipButton>
              <ToggletipContent className="normal-case">
                message spoken by this node before continuing.
              </ToggletipContent>
            </Toggletip>
          </div>
          <div className="border border-border-subtle bg-surface p-4 ">
            <div className="line-clamp-3 text-sm leading-5 text-foreground ">
              {messagePreview || 'No message configured'}
            </div>
          </div>
        </div>
      </Tile>
    </>
  );
}

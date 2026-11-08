import { Fragment } from 'react';
import { Information } from '@carbon/icons-react';
import { Metadata } from '@rapidaai/react';
import {
  Tag,
  Toggletip,
  ToggletipButton,
  ToggletipContent,
} from '@carbon/react';
import { getDisconnectReasonDisplay } from './disconnect-reason';

export const DisconnectReasonIndicator = ({
  reason,
  status,
  metadata,
  showLabel = true,
}: {
  reason: string;
  status?: string;
  metadata?: Metadata[];
  showLabel?: boolean;
}) => {
  const display = getDisconnectReasonDisplay(reason, status, metadata);

  return (
    <div className="inline-flex items-center gap-1 whitespace-nowrap">
      {showLabel && (
        <Tag size="md" type="gray">
          {display.label}
        </Tag>
      )}
      <Toggletip align="bottom-left">
        <ToggletipButton
          label="View disconnect details"
          title="View disconnect details"
        >
          <Information size={14} className="text-muted " />
        </ToggletipButton>
        <ToggletipContent>
          <div className="grid max-w-[360px] grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 whitespace-normal text-xs">
            {display.details.map(row => (
              <Fragment key={row.label}>
                <div className="whitespace-nowrap">{row.label}</div>
                <div className="break-words font-mono">{row.value}</div>
              </Fragment>
            ))}
          </div>
        </ToggletipContent>
      </Toggletip>
    </div>
  );
};

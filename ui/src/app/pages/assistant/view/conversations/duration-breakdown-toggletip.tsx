import { Fragment } from 'react';
import { Information } from '@carbon/icons-react';
import { Toggletip, ToggletipButton, ToggletipContent } from '@carbon/react';
import { AssistantConversation } from '@rapidaai/react';
import {
  getDurationBreakdownRows,
  UNKNOWN_DURATION_VALUE,
} from './session-list.helpers';

export const DurationBreakdownToggletip = ({
  conversation,
}: {
  conversation: AssistantConversation;
}) => {
  const rows = getDurationBreakdownRows(conversation);

  return (
    <Toggletip align="bottom-left">
      <ToggletipButton
        label="View duration breakdown"
        title="View duration breakdown"
      >
        <Information size={14} className="text-muted " />
      </ToggletipButton>
      <ToggletipContent>
        <div className="grid grid-cols-[auto_auto] gap-x-4 gap-y-1 text-xs">
          {rows.map(row => (
            <Fragment key={row.key}>
              <div className="whitespace-nowrap font-mono">{row.key}</div>
              <div className="whitespace-nowrap text-right tabular-nums">
                {row.value === UNKNOWN_DURATION_VALUE
                  ? row.value
                  : `${row.value}s`}
              </div>
            </Fragment>
          ))}
        </div>
      </ToggletipContent>
    </Toggletip>
  );
};

import { Information } from '@carbon/icons-react';
import { Toggletip, ToggletipButton, ToggletipContent } from '@carbon/react';
import type { ReactNode } from 'react';

interface HelpToggletipProps {
  label: string;
  helpText?: ReactNode;
}

export function HelpToggletip({ label, helpText }: HelpToggletipProps) {
  if (!helpText) return null;

  return (
    <Toggletip align="right">
      <ToggletipButton label={`${label} information`}>
        <Information size={14} />
      </ToggletipButton>
      <ToggletipContent>{helpText}</ToggletipContent>
    </Toggletip>
  );
}

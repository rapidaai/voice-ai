import { BorderButton } from '@/app/components/Form/Button';
import { CopyIcon } from '@/app/components/Icon/Copy';
import { TickIcon } from '@/app/components/Icon/Tick';
import { TD } from '@/app/components/Table/TD';
import React, { useState } from 'react';

export function CopyableColumn(props: { children: string }) {
  const [isChecked, setIsChecked] = useState(false);
  const copyItem = (item: string) => {
    setIsChecked(true);
    navigator.clipboard.writeText(item);
    setTimeout(() => {
      setIsChecked(false);
    }, 2000); // Reset back after 2 seconds
  };
  return (
    <TD>
      <div className="flex items-center justify-between group w-fit space-x-2">
        <div className="text-[15px] font-medium">{props.children}</div>
        <div className="flex items-start gap-1.5 opacity-0 transition-all focus-within:opacity-100 group-hover:opacity-100 [&:has([data-state='open'])]:opacity-100">
          <BorderButton
            className="h-6 w-6 p-0.5 border-[0.2px]"
            onClick={() => {
              copyItem(props.children);
            }}
          >
            {isChecked ? (
              <TickIcon className="w-4 h-4 text-green-600" />
            ) : (
              <CopyIcon className="w-4 h-4" />
            )}
          </BorderButton>
        </div>
      </div>
    </TD>
  );
}

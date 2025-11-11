import {
  BorderButton,
  ButtonProps,
  IButton,
} from '@/app/components/Form/Button';
import { CopyIcon } from '@/app/components/Icon/Copy';
import { TickIcon } from '@/app/components/Icon/Tick';
import { cn } from '@/styles/media';
import React, { useState } from 'react';

export function CopyButton(props: ButtonProps) {
  const [isChecked, setIsChecked] = useState(false);

  const copyItem = (item: string) => {
    setIsChecked(true);
    navigator.clipboard.writeText(item);
    setTimeout(() => {
      setIsChecked(false);
    }, 2000); // Reset back after 2 seconds
  };
  return (
    <IButton
      className={cn('h-6 w-6 p-0.5 border-[0.2px]', props.className)}
      onClick={() => {
        copyItem(props.children);
      }}
    >
      {isChecked ? (
        <TickIcon className="w-4 h-4 text-green-600" />
      ) : (
        <CopyIcon className="w-4 h-4" />
      )}{' '}
    </IButton>
  );
}

import React from 'react';
import { BorderButton, ButtonProps } from '@/app/components/Form/Button';

import { GuideIcon } from '@/app/components/Icon/guide';

interface GuideButtonProps extends ButtonProps {}
export function GuideButton(props: GuideButtonProps) {
  return (
    <BorderButton className="px-2" {...props}>
      <GuideIcon className="w-5 h-5" />
    </BorderButton>
  );
}

GuideButton.defaultProps = {
  type: 'button',
};

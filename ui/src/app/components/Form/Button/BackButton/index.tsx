import React from 'react';
import { BorderButton, ButtonProps } from '@/app/components/Form/Button';
import { BackIcon } from '@/app/components/Icon/Back';

interface BackButtonProps extends ButtonProps {}
export function BackButton(props: BackButtonProps) {
  return (
    <BorderButton {...props}>
      <BackIcon className="w-4 h-4" />
    </BorderButton>
  );
}

BackButton.defaultProps = {
  type: 'button',
};

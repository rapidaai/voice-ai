import React from 'react';
import { BorderButton, ButtonProps } from '@/app/components/form/button';
import { ReloadIcon } from '@/app/components/Icon/Reload';
import { Spinner } from '@/app/components/loader/spinner';

interface ReloadButtonProps extends ButtonProps {}
export function ReloadButton(props: ReloadButtonProps) {
  const { isLoading, ...attr } = props;
  return (
    <BorderButton {...attr}>
      {isLoading ? <Spinner /> : <ReloadIcon className="w-4 h-4" />}
    </BorderButton>
  );
}

ReloadButton.defaultProps = {
  type: 'button',
};

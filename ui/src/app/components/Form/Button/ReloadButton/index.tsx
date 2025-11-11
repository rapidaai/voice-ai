import React from 'react';
import { BorderButton, ButtonProps } from '@/app/components/Form/Button';
import { ReloadIcon } from '@/app/components/Icon/Reload';
import { Spinner } from '@/app/components/Loader/Spinner';

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

import { GreenNoticeBlock } from '@/app/components/container/message/notice-block';
import React, { FC, HTMLAttributes } from 'react';

interface SuccessMessageProps extends HTMLAttributes<HTMLDivElement> {
  message?: string;
}

export const SuccessMessage: FC<SuccessMessageProps> = (
  props: SuccessMessageProps,
) => {
  if (!props.message) return <></>;
  return <GreenNoticeBlock>{props.message}</GreenNoticeBlock>;
};

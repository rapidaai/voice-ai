import { RedNoticeBlock } from '@/app/components/container/message/notice-block';
import React, { FC, HTMLAttributes } from 'react';

interface ErrorMessageProps extends HTMLAttributes<HTMLDivElement> {
  message?: string;
}

export const ErrorMessage: FC<ErrorMessageProps> = (
  props: ErrorMessageProps,
) => {
  if (!props.message) return <></>;
  return <RedNoticeBlock>{props.message}</RedNoticeBlock>;
};

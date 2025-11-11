import type { FC } from 'react';
import React, { useCallback } from 'react';
import { useBoolean } from 'ahooks';
import Base from './base-editor';
import { Textarea } from '@/app/components/Form/Textarea';

type Props = {
  value: string;
  onChange: (value: string) => void;
  leftAction?: (boolean) => JSX.Element;
  rightAction?: (boolean) => JSX.Element;
  minHeight?: number;
  onBlur?: () => void;
  placeholder?: string;
  readonly?: boolean;
};

const TextEditor: FC<Props> = ({
  value,
  onChange,
  leftAction,
  rightAction,
  minHeight,
  onBlur,
  placeholder,
  readonly,
}) => {
  const [isFocus, { setTrue: setIsFocus, setFalse: setIsNotFocus }] =
    useBoolean(false);

  const handleBlur = useCallback(() => {
    setIsNotFocus();
    onBlur?.();
  }, [setIsNotFocus, onBlur]);

  return (
    <Base
      value={value}
      isFocus={isFocus}
      minHeight={minHeight}
      leftAction={leftAction}
      rightAction={rightAction}
      className="rounded-lg!"
    >
      <Textarea
        value={value}
        onChange={e => onChange(e.target.value)}
        onFocus={setIsFocus}
        onBlur={handleBlur}
        className="border-transparent! bg-transparent! h-full! w-full!"
        placeholder={placeholder}
        readOnly={readonly}
      />
    </Base>
  );
};
export default React.memo(TextEditor);

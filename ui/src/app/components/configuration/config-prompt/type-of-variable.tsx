import { Select } from '@/app/components/Form/Select';
import { cn } from '@/styles/media';
import { InputVarType } from '@/models/common';
import React, { FC } from 'react';

/**
 *
 * @param props
 * @returns
 */
export const TypeOfVariable: FC<
  React.SelectHTMLAttributes<HTMLSelectElement> & {
    type: string;
    onChange: (type: string) => void;
    allType: InputVarType[];
  }
> = React.memo(({ type, onChange, allType, className }) => {
  return (
    <Select
      value={type}
      placeholder="Select type of variable"
      options={allType.map(x => {
        return { name: x, value: x };
      })}
      className={cn('capitalize', className)}
      onChange={v => onChange(v.currentTarget.value)}
    />
  );
});

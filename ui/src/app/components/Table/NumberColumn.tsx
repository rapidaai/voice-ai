import { TD } from '@/app/components/Table/TD';
import { formatHumanReadableNumber } from '@/utils/format';
import React, { useEffect, useState } from 'react';

/**
 *
 * @param props
 * @returns
 */
export function NumberColumn(props: { num?: string }) {
  const [num, setNum] = useState('');

  useEffect(() => {
    if (!props.num) {
      setNum('0');
      return;
    }
    setNum(formatHumanReadableNumber(props.num));
  }, [props.num]);
  return (
    <TD className="text-center">
      <span className="font-medium max-w-[20rem] truncate">{num}</span>
    </TD>
  );
}

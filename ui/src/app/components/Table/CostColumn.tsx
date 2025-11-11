import { TD } from '@/app/components/Table/TD';
import React, { useEffect, useState } from 'react';

/**
 *
 * @param props
 * @returns
 */
export function CostColumn(props: { cost?: number }) {
  const [cost, setCost] = useState('');

  useEffect(() => {
    if (!props.cost) {
      setCost('0');
      return;
    }
    setCost(
      new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: 'USD',
      }).format(props.cost),
    );
  }, [props.cost]);
  return (
    <TD className="text-center">
      <span className="font-medium max-w-[20rem] truncate">{cost}</span>
    </TD>
  );
}

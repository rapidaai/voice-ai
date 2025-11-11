import React from 'react';
import { OptionMenu } from '@/app/components/Menu';
import { OptionMenuItem } from '@/app/components/Menu/index';

/**
 *
 * @param props
 * @returns
 */
export function UserOption(props: { id: string }) {
  return (
    <OptionMenu
      options={[
        {
          option: 'Edit user',
          onActionClick: () => {},
        },
        {
          option: <OptionMenuItem type="danger">Delete</OptionMenuItem>,
          onActionClick: () => {},
        },
      ]}
    />
  );
}

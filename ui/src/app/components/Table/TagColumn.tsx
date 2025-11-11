import { MultiplePills } from '@/app/components/Pill';
import { TD } from '@/app/components/Table/Body';
import React from 'react';

export function TagColumn(props: { tags?: string[] }) {
  return (
    <TD className="flex">
      {props.tags && props.tags.length > 0 ? (
        <MultiplePills tags={props.tags} />
      ) : (
        <>no tags</>
      )}
    </TD>
  );
}

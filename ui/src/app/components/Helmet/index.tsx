import { useWorkspace } from '@/context/workplace-context';
import React from 'react';
import { Helmet as HM } from 'react-helmet-async';

/**
 *
 */
interface HelmetProps {
  title?: string;
  meta?: { name: string; content: string }[];
}

/**
 *
 * @param props
 * @returns
 */
export function Helmet(props: HelmetProps) {
  const workspace = useWorkspace();
  return (
    <HM>
      <title>
        {props.title} - {workspace.domain}
      </title>
      {props.meta &&
        props.meta.map((mt, idx) => {
          return (
            <meta key={`meta_${idx}`} name={mt.name} content={mt.content} />
          );
        })}
    </HM>
  );
}

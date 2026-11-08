import React, { HTMLAttributes, useEffect, useState } from 'react';
import { cn } from '@/utils';
import { Tooltip as FloatingTooltip } from './floating-tooltip';

export function Pill(props: HTMLAttributes<HTMLSpanElement>) {
  return (
    <span
      onClick={props.onClick}
      className={cn(
        'px-3 py-1.5 font-medium truncate',
        'items-center text-[12px] capitalize bg-layer/30 rounded-[2px]',
        'ring-border-strong/20 ring-[1px]',
        props.className,
      )}
    >
      {props.children}
    </span>
  );
}

export function MultiplePills(props: {
  tags: string[] | undefined;
  items?: number;
  className?: string;
}) {
  const [pills, setPills] = useState<React.ReactElement[]>([]);
  useEffect(() => {
    if (!props.tags) return;
    let pl: React.ReactElement[] = [];
    for (let i = 0; i < props.tags.length; i++) {
      if (i > (props.items ? props.items : 1)) {
        pl.push(
          <FloatingTooltip
            key={`td_${i}_${props.tags[i]}`}
            content={
              <div className="w-fit">
                {props.tags.slice(i).map((s, i) => {
                  return <p key={i}>{s}</p>;
                })}
              </div>
            }
          >
            <Pill
              className={cn(props.className, 'block')}
              key={`td_${i}_${props.tags[i]}`}
            >
              {props.tags.length - (props.items ? props.items : 2)} +
            </Pill>
          </FloatingTooltip>,
        );
        break;
      }
      pl.push(
        <Pill className={props.className} key={`td_${i}_${props.tags[i]}`}>
          {props.tags[i]}
        </Pill>,
      );
    }
    setPills(pl);
  }, [props.tags, props.items]);

  return (
    <div className="flex items-center space-x-1.5 max-w-full">{pills}</div>
  );
}

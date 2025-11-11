import { cn } from '@/styles/media';
import { FC, HTMLAttributes, useEffect, useState } from 'react';
import { BUILDIN_TOOLS } from '@/app/components/tools';

interface ToolActionPillProps extends HTMLAttributes<HTMLSpanElement> {
  code: string;
}
export const ToolActionPill: FC<ToolActionPillProps> = props => {
  const [buildinTool, setBuildinTool] = useState<{
    name: string;
    icon: string;
  } | null>(null);
  useEffect(() => {
    if (props.code) {
      let cModel = BUILDIN_TOOLS.find(x => x.code === props.code);
      if (cModel) setBuildinTool(cModel);
    }
  }, [props.code]);

  return (
    <span
      onClick={props.onClick}
      className={cn(
        'px-2 py-1 truncate',
        'items-center bg-blue-400/10 dark:bg-blue-800/10 rounded-[2px]',
        'ring-blue-500/30 dark:ring-blue-400/20 ring-[0.7px]',
        'flex items-center justify-center w-fit',
        props.className,
      )}
    >
      <img
        alt={buildinTool?.name}
        src={buildinTool?.icon}
        className="w-4 h-4 mr-1.5 inline-block"
      />
      <span className="font-medium opacity-80 ">{buildinTool?.name}</span>
    </span>
  );
};

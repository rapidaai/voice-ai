import { ToolProvider } from '@rapidaai/react';
import { useProviderContext } from '@/context/provider-context';
import { cn } from '@/utils';
import { FC, HTMLAttributes, useEffect, useState } from 'react';

/**
 *
 */
interface ToolProviderPillProps extends HTMLAttributes<HTMLSpanElement> {
  toolProvider?: ToolProvider;
  toolProviderId?: string;
}

/**
 *
 * @param props
 * @returns
 */
export const ToolProviderPill: FC<ToolProviderPillProps> = props => {
  const { toolProviders } = useProviderContext();
  const [currentTool, setCurrentTool] = useState<ToolProvider | null>(
    props.toolProvider || null,
  );

  useEffect(() => {
    if (props.toolProviderId) {
      let cTool = toolProviders.find(x => x.getId() === props.toolProviderId);
      if (cTool) setCurrentTool(cTool);
    }
  }, [props.toolProviderId, toolProviders]);

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
        alt={currentTool?.getName()}
        src={currentTool?.getImage()}
        className="w-4 h-4 mr-1.5 inline-block"
      />
      <span className="font-medium opacity-80 ">{currentTool?.getName()}</span>
    </span>
  );
};

import { FC, HTMLAttributes } from 'react';
import { Card, CardDescription, CardTitle } from '@/app/components/base/cards';
import { cn } from '@/styles/media';
import { CardOptionMenu } from '@/app/components/Menu';
import { SquareFunction } from 'lucide-react';
import { AssistantTool } from '@rapidaai/react';
import { BUILDIN_TOOLS } from '@/app/components/tools';

interface ToolCardProps extends HTMLAttributes<HTMLDivElement> {
  tool: AssistantTool;
  options?: { option: any; onActionClick: () => void }[];
  iconClass?: string; // Fixed typo from 'iconClasss'
  titleClass?: string;
  isConnected?: boolean;
}

export const SelectToolCard: FC<ToolCardProps> = ({
  tool,
  options,
  className,
}) => {
  return (
    <Card className={cn(className)}>
      <header className="flex justify-between">
        <img
          alt={
            BUILDIN_TOOLS.find(x => x.code === tool.getExecutionmethod())?.name
          }
          src={
            BUILDIN_TOOLS?.find(x => x.code === tool.getExecutionmethod())?.icon
          }
          className="w-7 h-7 mr-1.5 inline-block"
        />
        {options && (
          <CardOptionMenu
            options={options}
            classNames="h-8 w-8 p-1 opacity-60"
          />
        )}
      </header>
      <div className="flex-1 mt-3">
        <CardTitle>{tool.getName()}</CardTitle>
        <CardDescription>{tool.getDescription()}</CardDescription>
      </div>
    </Card>
  );
};

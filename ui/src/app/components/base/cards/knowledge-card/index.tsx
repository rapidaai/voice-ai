import { FC, HTMLAttributes } from 'react';
import {
  Card,
  CardDescription,
  CardTitle,
  ClickableCard,
} from '@/app/components/base/cards';
import { Knowledge } from '@rapidaai/react';
import { KnowledgeIcon } from '@/app/components/Icon/knowledge';
import { cn } from '@/styles/media';
import { CardOptionMenu } from '@/app/components/Menu';
import { formatHumanReadableNumber } from '@/utils/format';

interface KnowledgeCardProps extends HTMLAttributes<HTMLDivElement> {
  knowledge: Knowledge;
  knowledgeOptions?: { option: any; onActionClick: () => void }[];
  iconClasss?: string;
  titleClass?: string;
  descriptionClass?: string;
}

export const SelectKnowledgeCard: FC<KnowledgeCardProps> = ({
  knowledge,
  knowledgeOptions,
  className,
}) => {
  return (
    <Card className={cn('p-0 rounded-[2px]', className)}>
      <div className="p-4 flex-1 flex flex-col">
        <header className="flex justify-between">
          <KnowledgeIcon className="w-7 h-7" strokeWidth={1.5} />
          {knowledgeOptions && (
            <CardOptionMenu
              options={knowledgeOptions}
              classNames="h-8 w-8 p-1 opacity-60"
            />
          )}
        </header>
        <div className="flex-1 mt-3">
          <CardTitle>{knowledge.getName()}</CardTitle>
          <CardDescription>{knowledge.getDescription()}</CardDescription>
        </div>
      </div>
    </Card>
  );
};

export const ClickableKnowledgeCard: FC<KnowledgeCardProps> = ({
  knowledge,
  className,
}) => {
  return (
    <ClickableCard
      to={`/knowledge/${knowledge.getId()}`}
      className={cn('p-0 rounded-[2px]', className)}
    >
      <div className="p-4">
        <header className="flex justify-between">
          <KnowledgeIcon className="w-7 h-7" strokeWidth={1.5} />
        </header>
        <div className="flex-1 mt-3">
          <CardTitle className="line-clamp-1">{knowledge.getName()}</CardTitle>
          <CardDescription className="line-clamp-1">
            {knowledge.getDescription()}
          </CardDescription>
        </div>
      </div>
      <div className="flex items-center text-xs opacity-80 mt-1 border-t px-4 justify-between">
        <div className="py-3">
          {formatHumanReadableNumber(knowledge.getDocumentcount())} docs
        </div>
        <div className="w-px h-full" />
        <div className="py-3">
          {formatHumanReadableNumber(knowledge.getWordcount())} words
        </div>
        <div className="w-px h-full" />
        <div className="py-3">
          {formatHumanReadableNumber(knowledge.getTokencount())} tokens
        </div>
      </div>
    </ClickableCard>
  );
};

import type { HTMLAttributes, ReactNode } from 'react';
import {
  Card,
  CardDescription,
  CardTitle,
  LinkCard,
} from '@/app/components/ui/primitives';
import type { Knowledge } from '@rapidaai/react';
import { cn } from '@/utils';
import { CardOptionMenu } from '@/app/components/ui/composites';
import { formatHumanReadableNumber } from '@/utils/format';
import { Folders } from '@carbon/icons-react';

interface KnowledgeCardProps extends HTMLAttributes<HTMLDivElement> {
  knowledge: Knowledge;
  knowledgeOptions?: { option: ReactNode; onActionClick: () => void }[];
  iconClasss?: string;
  titleClass?: string;
  descriptionClass?: string;
}

export function SelectKnowledgeCard({
  knowledge,
  knowledgeOptions,
  className,
}: KnowledgeCardProps) {
  return (
    <Card className={cn('p-0 rounded-[2px]', className)}>
      <div className="p-4 flex-1 flex flex-col">
        <header className="flex justify-between">
          <Folders className="w-7 h-7" />
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
}

export function ClickableKnowledgeCard({
  knowledge,
  className,
}: KnowledgeCardProps) {
  return (
    <LinkCard to={`/knowledge/${knowledge.getId()}`} className={className}>
      <div className="p-4 md:p-5 flex-1 flex flex-col">
        <header>
          <Folders className="w-7 h-7" />
        </header>
        <div className="flex-1 mt-3">
          <CardTitle className="line-clamp-1">{knowledge.getName()}</CardTitle>
          <CardDescription className="line-clamp-1">
            {knowledge.getDescription()}
          </CardDescription>
        </div>
      </div>
      <div className="flex items-center text-xs text-muted border-t border-border-subtle px-4 divide-x divide-border-subtle ">
        <div className="py-3 pr-4">
          {formatHumanReadableNumber(knowledge.getDocumentcount())} docs
        </div>
        <div className="py-3 px-4">
          {formatHumanReadableNumber(knowledge.getWordcount())} words
        </div>
        <div className="py-3 pl-4">
          {formatHumanReadableNumber(knowledge.getTokencount())} tokens
        </div>
      </div>
    </LinkCard>
  );
}

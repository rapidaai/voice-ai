import { Knowledge } from '@rapidaai/react';
import { Dropdown } from '@/app/components/dropdown';
import { FormLabel } from '@/app/components/form-label';
import { IButton, ILinkBorderButton } from '@/app/components/form/button';
import { FieldSet } from '@/app/components/form/fieldset';
import { useCredential } from '@/hooks';
import { useKnowledgePageStore } from '@/hooks/use-knowledge-page-store';
import { cn } from '@/utils';

import { ExternalLink, RotateCcw } from 'lucide-react';
import { FC, useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';

interface KnowledgeDropdownProps {
  className?: string;
  currentKnowledge?: string;
  onChangeKnowledge?: (k: Knowledge) => void;
}

export const KnowledgeDropdown: FC<KnowledgeDropdownProps> = props => {
  const [userId, token, projectId] = useCredential();
  const knowledgeActions = useKnowledgePageStore();
  const [loading, setLoading] = useState(false);

  const showLoader = () => setLoading(true);
  const hideLoader = () => setLoading(false);

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: Knowledge[]) => {
    hideLoader();
  }, []);
  /**
   * call the api
   */
  const getKnowledges = useCallback((projectId, token, userId) => {
    showLoader();
    knowledgeActions.getAllKnowledge(
      projectId,
      token,
      userId,
      onError,
      onSuccess,
    );
  }, []);

  /**
   *
   */
  useEffect(() => {
    if (props.currentKnowledge) {
      knowledgeActions.addCriteria('id', props.currentKnowledge, 'or');
    }
    getKnowledges(projectId, token, userId);
  }, [
    projectId,
    knowledgeActions.page,
    knowledgeActions.pageSize,
    JSON.stringify(knowledgeActions.criteria),
    props.currentKnowledge,
  ]);

  return (
    <FieldSet>
      <FormLabel>Knowledge</FormLabel>
      <div
        className={cn(
          'outline-solid outline-transparent',
          'focus-within:outline-blue-600 focus:outline-blue-600',
          'border-b border-gray-400 dark:border-gray-600',
          'focus-within:border-transparent!',
          'transition-all duration-200 ease-in-out',
          'flex relative',
          'bg-light-background dark:bg-gray-950',
          'pt-px pl-px',
          props.className,
        )}
      >
        <div className="w-full relative p-px pb-0">
          <Dropdown
            searchable
            className=" max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none! outline-hidden"
            currentValue={knowledgeActions.knowledgeBases.find(
              x => x.getId() === props.currentKnowledge,
            )}
            setValue={(c: Knowledge) => {
              if (props.onChangeKnowledge) props.onChangeKnowledge(c);
            }}
            onSearching={q => {
              if (q.target.value && q.target.value.trim() != '') {
                knowledgeActions.addCriteria('name', q.target.value, 'like');
              } else {
                knowledgeActions.removeCriteria('name');
              }
            }}
            allValue={knowledgeActions.knowledgeBases}
            placeholder="Select knowledge"
            option={(c: Knowledge) => {
              return (
                <div className="relative overflow-hidden flex-1 flex flex-row space-x-3">
                  <div className="flex ">
                    <span className="opacity-70">Knowledge</span>
                    <span className="opacity-70 px-1">/</span>
                    <span className="font-medium text-[14px]">
                      {c.getName()}
                    </span>
                    <span className="font-medium text-[14px] ml-4">
                      [{c.getId()}]
                    </span>
                  </div>
                </div>
              );
            }}
            label={c => {
              return (
                <div className="relative overflow-hidden flex-1 flex flex-row space-x-3">
                  <div className="flex">
                    <span className="opacity-70">Knowledge</span>
                    <span className="opacity-70 px-1">/</span>
                    <span className="font-medium text-[14px]">
                      {c.getName()}
                    </span>
                  </div>
                </div>
              );
            }}
          />
        </div>
        <IButton
          onClick={() => {
            getKnowledges(projectId, token, userId);
          }}
        >
          <RotateCcw className={cn('w-4 h-4')} strokeWidth={1.5} />
        </IButton>
        <ILinkBorderButton href="/knowledge/create-knowledge" target="_blank">
          <ExternalLink className={cn('w-4 h-4')} strokeWidth={1.5} />
        </ILinkBorderButton>
      </div>
    </FieldSet>
  );
};

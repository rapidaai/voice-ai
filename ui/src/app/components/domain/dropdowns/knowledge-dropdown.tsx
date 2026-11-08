import type { Knowledge } from '@rapidaai/react';
import { useCredential } from '@/hooks/use-credential';
import { useKnowledgePageStore } from '@/stores/knowledge/knowledge.store';
import { Launch, Renew } from '@carbon/icons-react';
import { useCallback, useEffect, useState } from 'react';
import toast from 'react-hot-toast/headless';
import { Dropdown, Button } from '@carbon/react';
import { cn } from '@/utils';

interface KnowledgeDropdownProps {
  className?: string;
  currentKnowledge?: string;
  onChangeKnowledge?: (k: Knowledge) => void;
}

interface KnowledgeDropdownViewProps extends KnowledgeDropdownProps {
  isLoading?: boolean;
  knowledgeBases: Knowledge[];
  onCreateKnowledge: () => void;
  onRefresh: () => void;
}

export function KnowledgeDropdownView({
  className,
  currentKnowledge,
  isLoading = false,
  knowledgeBases,
  onChangeKnowledge,
  onCreateKnowledge,
  onRefresh,
}: KnowledgeDropdownViewProps) {
  const selectedItem =
    knowledgeBases.find(knowledge => knowledge.getId() === currentKnowledge) ||
    null;

  return (
    <div className={cn(className)}>
      <div className="cds--label domain-connected-dropdown-label">
        Knowledge
      </div>
      <div className="domain-connected-dropdown-row flex w-full items-stretch border-b border-b-border-strong bg-field">
        <div className="min-w-0 flex-1">
          <Dropdown
            id="knowledge-dropdown"
            titleText="Knowledge"
            hideLabel
            label="Select knowledge"
            items={knowledgeBases}
            selectedItem={selectedItem}
            disabled={isLoading}
            itemToString={(item: Knowledge | null) =>
              item ? `${item.getName()} [${item.getId()}]` : ''
            }
            onChange={({ selectedItem }) => {
              if (selectedItem) {
                onChangeKnowledge?.(selectedItem);
              }
            }}
          />
        </div>
        <Button
          hasIconOnly
          renderIcon={Renew}
          iconDescription="Refresh knowledge"
          kind="ghost"
          size="md"
          disabled={isLoading}
          onClick={onRefresh}
          className="domain-connected-dropdown-action shrink-0"
        />
        <Button
          hasIconOnly
          renderIcon={Launch}
          iconDescription="Create knowledge"
          kind="ghost"
          size="md"
          onClick={onCreateKnowledge}
          className="domain-connected-dropdown-action shrink-0"
        />
      </div>
    </div>
  );
}

export function KnowledgeDropdown(props: KnowledgeDropdownProps) {
  const [userId, token, projectId] = useCredential();
  const knowledgeActions = useKnowledgePageStore();
  const [isLoading, setIsLoading] = useState(false);

  const showLoader = () => setIsLoading(true);
  const hideLoader = () => setIsLoading(false);

  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);

  const onSuccess = useCallback((data: Knowledge[]) => {
    hideLoader();
  }, []);

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
    <KnowledgeDropdownView
      className={props.className}
      currentKnowledge={props.currentKnowledge}
      isLoading={isLoading}
      knowledgeBases={knowledgeActions.knowledgeBases}
      onChangeKnowledge={props.onChangeKnowledge}
      onRefresh={() => getKnowledges(projectId, token, userId)}
      onCreateKnowledge={() =>
        window.open('/knowledge/create-knowledge', '_blank')
      }
    />
  );
}

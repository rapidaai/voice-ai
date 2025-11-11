import type { FC } from 'react';
import React, { useCallback, useEffect, useState } from 'react';
import { IBlueButton, ICancelButton } from '@/app/components/Form/Button';
import { useCredential, useRapidaStore } from '@/hooks';
import { useKnowledgePageStore } from '@/hooks/use-knowledge-page-store';
import { useNavigate } from 'react-router-dom';
import toast from 'react-hot-toast/headless';
import CheckboxCard from '@/app/components/Form/checkbox-card';
import { Knowledge } from '@rapidaai/react';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { SelectKnowledgeCard } from '@/app/components/base/cards/knowledge-card';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { MoveRight, Plus, X } from 'lucide-react';
import { IBlueBGButton } from '../../../Form/Button/index';
import { cn } from '@/styles/media';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import ModalFoot from '@/app/components/configuration/config-var/modal-foot';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';

/**
 *
 */
export type ISelectDataSetProps = {
  isShow: boolean;
  onClose: () => void;
  selectedIds: string[];
  onSelect: (dataSet: Knowledge[]) => void;
};

/**
 *
 * @param param0
 * @returns
 */
const SelectDataSet: FC<ISelectDataSetProps> = ({
  isShow,
  onClose,
  selectedIds,
  onSelect,
}) => {
  const [selected, setSelected] = React.useState<Knowledge[]>([]);
  const [userId, token, projectId] = useCredential();
  const knowledgeActions = useKnowledgePageStore();
  const { showLoader, hideLoader } = useRapidaStore();

  //   /
  const navigate = useNavigate();
  useEffect(() => {
    setSelected([
      ...knowledgeActions.knowledgeBases.filter(x =>
        selectedIds.some(y => x.getId() === y),
      ),
    ]);
  }, [selectedIds, knowledgeActions.knowledgeBases]);

  const [query, setQuery] = useState<string>('');
  /**
   *
   */
  useEffect(() => {
    if (query) {
      //   knowledgeActions.addCriteria('name', query, 'like');
    }
  }, [query]);

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

  useEffect(() => {
    getKnowledges(projectId, token, userId);
  }, [
    projectId,
    knowledgeActions.page,
    knowledgeActions.pageSize,
    knowledgeActions.criteria,
  ]);

  const toggleSelect = (dataSet: Knowledge) => {
    const isSelected = selected.some(item => item.getId() === dataSet.getId());
    if (isSelected) {
      setSelected(selected.filter(item => item.getId() !== dataSet.getId()));
    } else {
      setSelected([...selected, dataSet]);
    }
  };

  const handleSelect = () => {
    onSelect(selected);
  };
  return (
    <GenericModal modalOpen={isShow} setModalOpen={onClose}>
      <ModalFitHeightBlock>
        <ModalHeader onClose={onClose}>
          <ModalTitleBlock>Select Knowledge</ModalTitleBlock>
        </ModalHeader>
        <ModalBody className="py-0 px-0 h-[80dvh] overflow-auto space-y-0">
          <BluredWrapper className="sticky top-0 z-10 pr-0 py-2">
            <SearchIconInput
              className="text-sm h-8 space-x-2 w-full pl-7 bg-light-background"
              wrapperClassName="h-8 w-full"
              onChange={x => {
                setQuery(x.target.value);
              }}
            />
            <TablePagination
              currentPage={knowledgeActions.page}
              onChangeCurrentPage={knowledgeActions.setPage}
              totalItem={knowledgeActions.totalCount}
              pageSize={knowledgeActions.pageSize}
              onChangePageSize={knowledgeActions.setPageSize}
            />
            <div>
              <span className="flex items-center justify-center px-3 h-10">
                <span className="w-0.5 h-5 dark:bg-gray-700 bg-gray-300"></span>
              </span>
            </div>
            <IBlueButton
              onClick={() => {
                navigate('/knowledge/create-knowledge');
              }}
              className="shrink-0"
            >
              Add new knowledge
              <Plus className="w-4 h-4 ml-1.5"></Plus>
            </IBlueButton>
          </BluredWrapper>
          {knowledgeActions.knowledgeBases &&
          knowledgeActions.knowledgeBases?.length > 0 ? (
            <div className="overflow-y-auto grid-cols-2 gap-3 grid px-4 py-4">
              {knowledgeActions.knowledgeBases
                .filter(x => {
                  if (!query) return true;
                  return x
                    .getName()
                    .toLowerCase()
                    .includes(query.toLowerCase());
                })
                .map((item, idx) => (
                  <CheckboxCard
                    selectedClassNames="border border-blue-600/50"
                    key={`${idx}-checkbox-sd-kb`}
                    id={`${idx}-checkbox-sd-kb`}
                    name={`${idx}-checkbox-sd-kb`}
                    checked={selected.some(i => i.getId() === item.getId())}
                    type="checkbox"
                    onChange={() => {
                      toggleSelect(item);
                    }}
                  >
                    <SelectKnowledgeCard
                      knowledge={item}
                      className="shadow-sm max-w-none"
                    />
                  </CheckboxCard>
                ))}
            </div>
          ) : (
            <div className="px-2 py-2">
              <ActionableEmptyMessage
                title="No Knowledge"
                subtitle="There are no Knowledge created."
                action="Create new knowledge"
                onActionClick={() => {
                  navigate('/knowledge/create-knowledge');
                }}
              />
            </div>
          )}
        </ModalBody>

        <ModalFooter>
          <ICancelButton className="rounded-[2px] px-4" onClick={onClose}>
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="rounded-[2px] px-4"
            type="button"
            onClick={handleSelect}
          >
            Add knowledge
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};
export default SelectDataSet;

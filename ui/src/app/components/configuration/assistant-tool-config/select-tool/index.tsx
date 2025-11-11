import type { FC } from 'react';
import React, { useCallback, useEffect, useState } from 'react';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { useCredential, useRapidaStore } from '@/hooks';
import toast from 'react-hot-toast/headless';
import CheckboxCard from '@/app/components/Form/checkbox-card';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { SelectToolCard } from '@/app/components/base/cards/tool-card';
import { MoveRight } from 'lucide-react';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { AssistantTool } from '@rapidaai/react';

/**
 *
 */
export type SelectToolProps = {
  isShow: boolean;
  onClose: () => void;
  selectedIds: string[];
  onSelect: (tools: AssistantTool[]) => void;
};

/**
 *
 * @param param0
 * @returns
 */
export const SelectTool: FC<SelectToolProps> = ({
  isShow,
  onClose,
  selectedIds,
  onSelect,
}) => {
  return <></>;
  //   const [selected, setSelected] = React.useState<Tool[]>([]);
  //   const [userId, token, projectId] = useCredential();
  //   const toolAction = useToolPageStore();
  //   const { showLoader, hideLoader } = useRapidaStore();
  //   const { goToCreateKnowledge } = useGlobalNavigation();
  //   //   /
  //   useEffect(() => {
  //     setSelected([
  //       ...toolAction.tools.filter(x => selectedIds.some(y => x.getId() === y)),
  //     ]);
  //   }, [selectedIds, toolAction.tools]);

  //   const [query, setQuery] = useState<string>('');
  //   const onError = useCallback((err: string) => {
  //     hideLoader();
  //     toast.error(err);
  //   }, []);
  //   const onSuccess = useCallback((data: Tool[]) => {
  //     hideLoader();
  //   }, []);
  //   /**
  //    * call the api
  //    */
  //   const getTools = useCallback((projectId, token, userId) => {
  //     showLoader();
  //     toolAction.getAllTool(projectId, token, userId, onError, onSuccess);
  //   }, []);

  //   useEffect(() => {
  //     getTools(projectId, token, userId);
  //   }, [projectId, toolAction.page, toolAction.pageSize, toolAction.criteria]);

  //   const toggleSelect = (dataSet: Tool) => {
  //     const isSelected = selected.some(item => item.getId() === dataSet.getId());
  //     if (isSelected) {
  //       setSelected(selected.filter(item => item.getId() !== dataSet.getId()));
  //     } else {
  //       setSelected([...selected, dataSet]);
  //     }
  //   };

  //   const handleSelect = () => {
  //     onSelect(selected);
  //   };

  //   return (
  //     <GenericModal modalOpen={isShow} setModalOpen={onClose}>
  //       <ModalFitHeightBlock>
  //         <ModalHeader onClose={onClose}>
  //           <ModalTitleBlock>Select tools</ModalTitleBlock>
  //         </ModalHeader>
  //         <ModalBody className="py-0 px-0 h-[60dvh] overflow-auto space-y-0">
  //           <BluredWrapper className="sticky top-0 z-10 pr-0 py-2">
  //             <SearchIconInput
  //               className="text-sm h-8 space-x-2 w-full pl-7 bg-light-background"
  //               wrapperClassName="h-8 w-full"
  //               onChange={x => {
  //                 setQuery(x.target.value);
  //               }}
  //             />
  //             <TablePagination
  //               currentPage={toolAction.page}
  //               onChangeCurrentPage={toolAction.setPage}
  //               totalItem={toolAction.totalCount}
  //               pageSize={toolAction.pageSize}
  //               onChangePageSize={toolAction.setPageSize}
  //             />
  //           </BluredWrapper>
  //           {toolAction.tools && toolAction.tools?.length > 0 ? (
  //             <div className="overflow-y-auto grid-cols-3 grid px-4 gap-3 py-4">
  //               {toolAction.tools
  //                 .filter(x => {
  //                   if (!query) return true;
  //                   return x
  //                     .getName()
  //                     .toLowerCase()
  //                     .includes(query.toLowerCase());
  //                 })
  //                 .map((item, idx) => (
  //                   <CheckboxCard
  //                     selectedClassNames="border border-blue-600/50"
  //                     key={`${idx}-checkbox-sd-kb`}
  //                     id={`${idx}-checkbox-sd-kb`}
  //                     name={`${idx}-checkbox-sd-kb`}
  //                     checked={selected.some(i => i.getId() === item.getId())}
  //                     type="checkbox"
  //                     onChange={() => {
  //                       toggleSelect(item);
  //                     }}
  //                   >
  //                     <SelectToolCard tool={item} />
  //                   </CheckboxCard>
  //                 ))}
  //             </div>
  //           ) : (
  //             <div className="px-2 py-2">
  //               <ActionableEmptyMessage
  //                 title="No Skills"
  //                 subtitle="There are no Knowledge created."
  //                 action="Create new knowledge"
  //                 onActionClick={goToCreateKnowledge}
  //               />
  //             </div>
  //           )}
  //         </ModalBody>
  //         <ModalFooter>
  //           <ICancelButton className="px-4 rounded-[2px]" onClick={onClose}>
  //             Cancel
  //           </ICancelButton>
  //           <IBlueBGButton
  //             className="px-4 rounded-[2px]"
  //             type="button"
  //             onClick={handleSelect}
  //           >
  //             Add tool
  //           </IBlueBGButton>
  //         </ModalFooter>
  //       </ModalFitHeightBlock>
  //     </GenericModal>
  //   );
};

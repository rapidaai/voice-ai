import { RightSideModal } from '@/app/components/base/modal/right-side-modal';
import { ModalProps } from '@/app/components/base/modal';
import { FC, HTMLAttributes, memo, useState } from 'react';
import { FormLabel } from '@/app/components/form-label';
import { Input } from '@/app/components/Form/Input';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Datepicker } from '@/app/components/Datepicker';
import SourceSelector from '@/app/components/selectors/source';
import StatusSelector from '@/app/components/selectors/status';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { ModalBody } from '@/app/components/base/modal/modal-body';

interface AssistantTraceFilterDialogProps
  extends ModalProps,
    HTMLAttributes<HTMLDivElement> {
  filters: {
    dateFrom?: string;
    dateTo?: string;
    source?: string;
    sessionId?: string;
    status?: string;
  };
  onFiltersChange: (filters: {
    dateFrom?: string;
    dateTo?: string;
    source?: string;
    sessionId?: string;
    status?: string;
  }) => void;
}
export const AssistantTraceFilterDialog: FC<AssistantTraceFilterDialogProps> =
  memo(({ filters, onFiltersChange, ...mldAttr }) => {
    const [localFilters, setLocalFilters] = useState(filters);

    const updateLocalFilter = (key: string, value: any) => {
      setLocalFilters(prev => ({ ...prev, [key]: value }));
    };

    const handleApply = () => {
      onFiltersChange(localFilters);
      mldAttr.setModalOpen(false);
    };

    const handleDateSelect = (to: Date, from: Date) => {
      updateLocalFilter('dateFrom', from.toISOString());
      updateLocalFilter('dateTo', to.toISOString());
    };

    return (
      <RightSideModal
        {...mldAttr}
        title="Filter"
        className={'min-w-[30vw]! overflow-visible'}
      >
        <div className="relative overflow-auto flex flex-col flex-1 grow">
          <ModalBody>
            <FieldSet>
              <FormLabel className="normal-case">Date Range</FormLabel>
              <Datepicker
                align="right"
                className="bg-light-background"
                defaultDate={
                  localFilters.dateFrom && localFilters.dateTo
                    ? {
                        from: new Date(localFilters.dateFrom),
                        to: new Date(localFilters.dateTo),
                      }
                    : undefined
                }
                onDateSelect={handleDateSelect}
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Session ID</FormLabel>
              <Input
                type="text"
                placeholder="Enter session ID..."
                value={localFilters.sessionId || ''}
                onChange={e => updateLocalFilter('sessionId', e.target.value)}
                className="bg-light-background"
              />
            </FieldSet>
            {/* Source Filter */}
            <FieldSet>
              <FormLabel>Source</FormLabel>
              <SourceSelector
                selectedSource={localFilters.source}
                selectSource={v => {
                  updateLocalFilter('source', v);
                }}
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Status</FormLabel>
              <StatusSelector
                selectedStatus={localFilters.status}
                selectStatus={v => {
                  updateLocalFilter('status', v);
                }}
              />
            </FieldSet>
          </ModalBody>
          <ModalFooter className="sticky bottom-0">
            <ICancelButton
              className="px-4 rounded-[2px]"
              onClick={() => {
                mldAttr.setModalOpen(false);
              }}
            >
              Cancel
            </ICancelButton>
            <IBlueBGButton
              className="px-4 rounded-[2px]"
              type="button"
              onClick={handleApply}
            >
              Apply
            </IBlueBGButton>
          </ModalFooter>
        </div>
      </RightSideModal>
    );
  });

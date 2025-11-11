import { UpdateKnowledgeDocumentSegment } from '@rapidaai/react';
import { BaseResponse } from '@rapidaai/react';
import { KnowledgeDocumentSegment } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { GenericModal } from '@/app/components/base/modal';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { FormLabel } from '@/app/components/form-label';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { useCurrentCredential } from '@/hooks/use-credential';
import { Check } from 'lucide-react';
import { FC, useState } from 'react';
import { connectionConfig } from '@/configs';

/**
 *
 */
export const EditKnowledgeDocumentSegmentDialog: FC<{
  segment: KnowledgeDocumentSegment;
  onClose: () => void;
  onUpdate: () => void;
}> = ({ segment, onClose, onUpdate }) => {
  const { authId, token, projectId } = useCurrentCredential();
  const [entities, setEntities] = useState({
    documentName: segment?.getMetadata()?.getDocumentName() || '',
    organizations:
      segment.getEntities()?.getOrganizationsList()?.join(', ') || '',
    dates: segment.getEntities()?.getDatesList()?.join(', ') || '',
    products: segment.getEntities()?.getProductsList()?.join(', ') || '',
    events: segment.getEntities()?.getEventsList()?.join(', ') || '',
    industries: segment.getEntities()?.getIndustriesList()?.join(', ') || '',
    locations: segment.getEntities()?.getLocationsList()?.join(', ') || '',
    people: segment.getEntities()?.getPeopleList()?.join(', ') || '',
    times: segment.getEntities()?.getTimesList()?.join(', ') || '',
    quantities: segment.getEntities()?.getQuantitiesList()?.join(', ') || '',
  });

  const processEntity = (entityString: string) =>
    entityString
      .split(', ')
      .map(item => item.trim())
      .filter(item => item !== '');

  const handleUpdate = () => {
    UpdateKnowledgeDocumentSegment(
      connectionConfig,
      segment.getDocumentId(),
      segment.getIndex().toString(),
      processEntity(entities.organizations),
      processEntity(entities.dates),
      processEntity(entities.products),
      processEntity(entities.events),
      processEntity(entities.people),
      processEntity(entities.times),
      processEntity(entities.quantities),
      processEntity(entities.locations),
      processEntity(entities.industries),
      entities.documentName,
      (err: ServiceError | null, response: BaseResponse | null) => {
        if (err) {
          console.error('Error updating segment:', err);
        } else {
          onUpdate();
          onClose();
        }
      },
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': authId,
      },
    );
  };

  const handleEntityChange = (key: string, value: string) => {
    setEntities(prev => ({ ...prev, [key]: value }));
  };

  return (
    <GenericModal modalOpen={true} setModalOpen={onClose}>
      <ModalFitHeightBlock>
        <ModalHeader onClose={onClose}>
          <ModalTitleBlock>Edit Document Segment</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <div className="p-6 space-y-6 h-[80dvh] overflow-auto">
            <FieldSet>
              <FormLabel>Document Segment ID</FormLabel>
              <Input disabled type="text" value={segment.getDocumentId()} />
            </FieldSet>

            <FieldSet>
              <FormLabel>Document Name</FormLabel>
              <Input
                type="text"
                value={entities.documentName}
                onChange={e =>
                  handleEntityChange('documentName', e.target.value)
                }
                placeholder="Enter document names"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Organizations</FormLabel>
              <Input
                type="text"
                value={entities.organizations}
                onChange={e =>
                  handleEntityChange('organizations', e.target.value)
                }
                placeholder="Enter organizations separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Dates</FormLabel>
              <Input
                type="text"
                value={entities.dates}
                onChange={e => handleEntityChange('dates', e.target.value)}
                placeholder="Enter dates separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Products</FormLabel>
              <Input
                type="text"
                value={entities.products}
                onChange={e => handleEntityChange('products', e.target.value)}
                placeholder="Enter products separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Events</FormLabel>
              <Input
                type="text"
                value={entities.events}
                onChange={e => handleEntityChange('events', e.target.value)}
                placeholder="Enter events separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Industries</FormLabel>
              <Input
                type="text"
                value={entities.industries}
                onChange={e => handleEntityChange('industries', e.target.value)}
                placeholder="Enter industries separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Locations</FormLabel>
              <Input
                type="text"
                value={entities.locations}
                onChange={e => handleEntityChange('locations', e.target.value)}
                placeholder="Enter locations separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>People</FormLabel>
              <Input
                type="text"
                value={entities.people}
                onChange={e => handleEntityChange('people', e.target.value)}
                placeholder="Enter people's names separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Times</FormLabel>
              <Input
                type="text"
                value={entities.times}
                onChange={e => handleEntityChange('times', e.target.value)}
                placeholder="Enter times separated by commas"
              />
            </FieldSet>
            <FieldSet>
              <FormLabel>Quantities</FormLabel>
              <Input
                type="text"
                value={entities.quantities}
                onChange={e => handleEntityChange('quantities', e.target.value)}
                placeholder="Enter quantities separated by commas"
              />
            </FieldSet>
          </div>
        </ModalBody>

        <ModalFooter>
          <ICancelButton className="px-4 rounded-[2px]" onClick={onClose}>
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={handleUpdate}
          >
            Update document
            <Check className="ml-2" strokeWidth={1.5} />
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};

import { Tag } from '@rapidaai/react';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { TagInput } from '@/app/components/Form/tag-input';
import { KnowledgeTags } from '@/app/components/Form/tag-input/knowledge-tags';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { useRapidaStore } from '@/hooks';
import { MoveRight } from 'lucide-react';
import React, { FC, memo, useEffect, useState } from 'react';

// Define props interface for the CreateTagDialog component
interface CreateTagDialogProps extends ModalProps {
  title: string; // Title of the modal dialog
  tags?: string[]; // Optional initial tags
  allTags?: string[]; // Optional list of all tags
  onCreateTag: (
    tags: string[],
    onError: (err: string) => void,
    onSuccess: (e: Tag) => void,
  ) => void; // Callback function for creating tags
}

// Create the CreateTagDialog functional component
export const CreateTagDialog: FC<CreateTagDialogProps> = memo(
  ({ title, tags, allTags, onCreateTag, setModalOpen, modalOpen }) => {
    // State for error handling
    const [error, setError] = useState('');
    // State for managing tags
    const [_tags, _setTags] = useState<string[]>([]);
    // Access rapidaStore from hooks
    const rapidaStore = useRapidaStore();

    // Function to add a new tag
    const addTag = (tag: string) => {
      _setTags([..._tags, tag]);
    };

    // Function to remove a tag by index
    const removeTag = (index: number) => {
      const newTags = [..._tags];
      newTags.splice(index, 1);
      _setTags(newTags);
    };

    // Effect to initialize tags when the `tags` prop changes
    useEffect(() => {
      if (tags) _setTags(tags);
    }, [tags]);

    // Function to create tags and handle success/error scenarios
    const createTag = () => {
      rapidaStore.showLoader('overlay');
      onCreateTag(
        _tags,
        (err: string) => {
          rapidaStore.hideLoader();
          setError(err); // Set error message on failure
        },
        (rc: Tag) => {
          rapidaStore.hideLoader();
          setModalOpen(false); // Close modal on success
        },
      );
    };

    return (
      <GenericModal modalOpen={modalOpen} setModalOpen={setModalOpen}>
        <ModalFitHeightBlock>
          <ModalHeader
            onClose={() => {
              setModalOpen(false);
            }}
          >
            <ModalTitleBlock>{title}</ModalTitleBlock>
          </ModalHeader>
          <ModalBody>
            <div className="px-4 py-6">
              <TagInput
                tags={_tags}
                addTag={addTag}
                removeTag={removeTag}
                allTags={allTags ? allTags : KnowledgeTags} // Use KnowledgeTags as default if allTags is not provided
              />
              <ErrorMessage message={error} /> {/* Display error message */}
            </div>
          </ModalBody>
          <ModalFooter>
            <ICancelButton
              className="px-4 rounded-[2px]"
              onClick={() => {
                setModalOpen(false);
              }}
            >
              Cancel
            </ICancelButton>
            <IBlueBGButton
              className="px-4 rounded-[2px]"
              type="button"
              onClick={createTag}
              isLoading={rapidaStore.loading}
            >
              Update Tags
              <MoveRight className="ml-2" strokeWidth={1.5} />
            </IBlueBGButton>
          </ModalFooter>
        </ModalFitHeightBlock>
      </GenericModal>
    );
  },
);

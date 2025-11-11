import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalFormBlock } from '@/app/components/blocks/modal-form-block';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { Textarea } from '@/app/components/Form/Textarea';
import { Check } from 'lucide-react';
import { FC, useState } from 'react';

export const MessageFeedbackDialog: FC<
  ModalProps & { onSubmitFeedback: (feedback: string) => void }
> = props => {
  const [feedbackText, setFeedbackText] = useState('');
  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFormBlock>
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>What can be improved?</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          <div className="px-4 py-6">
            <p className="font-semibold text-base mt-1">
              Tell us what went wrong or how we can make this answer more
              helpful.
            </p>
            <div className="mt-4">
              <Textarea
                required
                rows={3}
                placeholder="Your feedback..."
                value={feedbackText}
                onChange={e => setFeedbackText(e.target.value)}
              />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => props.setModalOpen(false)}
          >
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={() => {
              props.setModalOpen(false);
              props.onSubmitFeedback(feedbackText);
            }}
            disabled={!feedbackText.trim()}
          >
            Submit feedback
            <Check className="ml-2" strokeWidth={1.5} />
          </IBlueBGButton>
        </ModalFooter>
      </ModalFormBlock>
    </GenericModal>
  );
};

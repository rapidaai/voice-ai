import React, { FC } from 'react';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { HowItWorks } from '@/app/components/base/modal/how-it-works-modal';
import { IBlueBGButton } from '@/app/components/Form/Button';
import { Check } from 'lucide-react';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';

/**
 * creation provider key dialog props that gives ability for opening and closing modal props
 */
interface HowAssistantWorksModalProps extends ModalProps {}
/**
 *
 * to create a provider key for given model
 * @param props
 * @returns
 */
export const HowAssistantWorksDialog: FC<
  HowAssistantWorksModalProps
> = props => {
  const steps = [
    {
      title: 'Design Your AI Assistant',
      icon: (
        <svg
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={1.5}
          stroke="currentColor"
          className="w-6 h-6"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M15.182 15.182a4.5 4.5 0 01-6.364 0M21 12a9 9 0 11-18 0 9 9 0 0118 0zM9.75 9.75c0 .414-.168.75-.375.75S9 10.164 9 9.75 9.168 9 9.375 9s.375.336.375.75zm-.375 0h.008v.015h-.008V9.75zm5.625 0c0 .414-.168.75-.375.75s-.375-.336-.375-.75.168-.75.375-.75.375.336.375.75zm-.375 0h.008v.015h-.008V9.75z"
          />
        </svg>
      ),
      description:
        "Let's start by imagining your perfect AI helper! Choose a Foundation model that fits your needs, then add Action groups to give your assistant its superpowers. It's like building a friendly robot companion!",
    },
    {
      title: 'Test and Refine',
      icon: (
        <svg
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={1.5}
          stroke="currentColor"
          className="w-6 h-6"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M10.5 6h9.75M10.5 6a1.5 1.5 0 11-3 0m3 0a1.5 1.5 0 10-3 0M3.75 6H7.5m3 12h9.75m-9.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-3.75 0H7.5m9-6h3.75m-3.75 0a1.5 1.5 0 01-3 0m3 0a1.5 1.5 0 00-3 0m-9.75 0h9.75"
          />
        </svg>
      ),
      description:
        "Time to play with your new AI friend! Chat with it, give it tasks, and see how it does. If it's not quite perfect, don't worry - we can tweak and improve it together until it's just right.",
    },
    {
      title: 'Deploy and Share',
      icon: (
        <svg
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
          strokeWidth={1.5}
          stroke="currentColor"
          className="w-6 h-6"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M9 8.25H7.5a2.25 2.25 0 00-2.25 2.25v9a2.25 2.25 0 002.25 2.25h9a2.25 2.25 0 002.25-2.25v-9a2.25 2.25 0 00-2.25-2.25H15M9 12l3 3m0 0l3-3m-3 3V2.25"
          />
        </svg>
      ),
      description:
        "Your AI assistant is ready to meet the world! Give it a cool name (an 'alias'), and send it out to help in your apps. Don't forget to keep an eye on how it's doing - it might surprise you with how much it can learn and grow!",
    },
  ];

  /**
   *
   */
  return (
    <GenericModal modalOpen={props.modalOpen} setModalOpen={props.setModalOpen}>
      <ModalFitHeightBlock className="w-1/2">
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
        >
          <ModalTitleBlock>How it works</ModalTitleBlock>
        </ModalHeader>
        <HowItWorks steps={steps} />
        <ModalFooter>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={() => {
              props.setModalOpen(false);
            }}
          >
            Got it
            <Check className="ml-2 w-4 h-5" strokeWidth={1.5} />
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};

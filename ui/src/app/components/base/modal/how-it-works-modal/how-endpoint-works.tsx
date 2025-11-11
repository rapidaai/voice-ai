import React, { FC } from 'react';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { HowItWorks } from '@/app/components/base/modal/how-it-works-modal';
import { IBlueBGButton } from '@/app/components/Form/Button';
import { Check } from 'lucide-react';
import { cn } from '@/styles/media';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';

/**
 * creation provider key dialog props that gives ability for opening and closing modal props
 */
interface HowEndpointWorksModalProps extends ModalProps {}
/**
 *
 * to create a provider key for given model
 * @param props
 * @returns
 */
export const HowEndpointWorksDialog: FC<HowEndpointWorksModalProps> = props => {
  const steps = [
    {
      title: 'Create an Endpoint',
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
            d="M8.625 9.75a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375m-13.5 3.01c0 1.6 1.123 2.994 2.707 3.227 1.087.16 2.185.283 3.293.369V21l4.184-4.183a1.14 1.14 0 0 1 .778-.332 48.294 48.294 0 0 0 5.83-.498c1.585-.233 2.708-1.626 2.708-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z"
          />
        </svg>
      ),
      description:
        'An endpoint is a combination of model and prompt that can be easily deployed for your application. Create an endpoint by configuring them for your specific use case.',
    },
    {
      title: 'Use the endpoint',
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
            d="m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9"
          />
        </svg>
      ),
      description:
        "Integrate the endpoint into your application. This allows you to easily leverage the power of the models and prompts you've configured without managing the underlying infrastructure.",
    },
    {
      title: 'Iterate and Optimize',
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
            d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"
          />
        </svg>
      ),
      description:
        "Refine your endpoint's performance without changing code or integration. Monitor results, gather feedback, and adjust configurations to enhance effectiveness for your use case.",
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
            <Check className="ml-2" strokeWidth={1.5} />
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};

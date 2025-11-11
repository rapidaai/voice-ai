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
interface HowKnowledgeWorksModalProps extends ModalProps {}
/**
 *
 * to create a provider key for given model
 * @param props
 * @returns
 */
export const HowKnowledgeWorksDialog: FC<
  HowKnowledgeWorksModalProps
> = props => {
  const steps = [
    {
      title: 'Upload and Explore',
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
            d="M9 8.25H7.5a2.25 2.25 0 00-2.25 2.25v9a2.25 2.25 0 002.25 2.25h9a2.25 2.25 0 002.25-2.25v-9a2.25 2.25 0 00-2.25-2.25H15m0-3l-3-3m0 0l-3 3m3-3V15"
          />
        </svg>
      ),
      description:
        "Upload documents from third-party sources or your company's private data. Instantly start AI-powered conversations about their contents, gaining quick insights without complex setups.",
    },
    {
      title: 'Build Knowledge Base',
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
            d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25"
          />
        </svg>
      ),
      description:
        'Create a powerful knowledge base by simply specifying your data sources. Our system automatically selects the best embedding model, configures storage, and handles syncing, eliminating infrastructure worries.',
    },
    {
      title: 'Deploy and Integrate',
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
            d="M3 8.688c0-.864.933-1.405 1.683-.977l7.108 4.062a1.125 1.125 0 010 1.953l-7.108 4.062A1.125 1.125 0 013 16.81V8.688zM12.75 8.688c0-.864.933-1.405 1.683-.977l7.108 4.062a1.125 1.125 0 010 1.953l-7.108 4.062a1.125 1.125 0 01-1.683-.977V8.688z"
          />
        </svg>
      ),
      description:
        'Seamlessly integrate your knowledge base into your assistant. Enjoy automated syncing and updates without worrying about infrastructure, keeping your AI capabilities always up-to-date.',
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

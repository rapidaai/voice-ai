import React, { FC, useEffect, useState } from 'react';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import {
  BuildinTool,
  BuildinToolConfig,
  GetDefaultToolConfigIfInvalid,
  GetDefaultToolDefintion,
  ValidateToolDefaultOptions,
} from '@/app/components/tools';
import { GenericModal, ModalProps } from '@/app/components/base/modal';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';

interface ConfigureAssistantToolDialogProps extends ModalProps {
  initialData: {
    name: string;
    description: string;
    fields: string;
    buildinToolConfig: BuildinToolConfig;
  } | null;
  onChange?: (data: {
    name: string;
    description: string;
    fields: string;
    buildinToolConfig: BuildinToolConfig;
  }) => void;
  onValidateConfig?: (data: {
    name: string;
    description: string;
    fields: string;
    buildinToolConfig: BuildinToolConfig;
  }) => string | null; // Return error message or null if valid
}

export const ConfigureAssistantToolDialog: FC<
  ConfigureAssistantToolDialogProps
> = props => {
  //
  const [toolDefinition, setToolDefinition] = useState<{
    name: string;
    description: string;
    parameters: string;
  }>(
    GetDefaultToolDefintion('knowledge_retrieval', {
      name: '',
      description: '',
      parameters: '',
    }),
  );

  //
  const [buildinToolConfig, setBuildinToolConfig] = useState<BuildinToolConfig>(
    {
      code: 'knowledge_retrieval',
      parameters: GetDefaultToolConfigIfInvalid('knowledge_retrieval', []),
    },
  );

  const [errorMessage, setErrorMessage] = useState('');
  const resetState = () => {
    setBuildinToolConfig({
      code: 'knowledge_retrieval',
      parameters: GetDefaultToolConfigIfInvalid('knowledge_retrieval', []),
    });
    setToolDefinition(
      GetDefaultToolDefintion('knowledge_retrieval', {
        name: '',
        description: '',
        parameters: '',
      }),
    );

    setErrorMessage('');
  };

  useEffect(() => {
    if (props.modalOpen && props.initialData) {
      setToolDefinition(
        GetDefaultToolDefintion(
          props.initialData.buildinToolConfig.code || 'knowledge_retrieval',
          {
            name: props.initialData.name || '',
            description: props.initialData.description || '',
            parameters: props.initialData.fields || '',
          },
        ),
      );
      setBuildinToolConfig(
        props.initialData.buildinToolConfig || {
          code: 'knowledge_retrieval',
          parameters: GetDefaultToolConfigIfInvalid('knowledge_retrieval', []),
        },
      );
    } else if (!props.modalOpen) {
      resetState();
    }
  }, [props.initialData, props.modalOpen]);

  const onChangeBuildinToolConfig = (code: string) => {
    setBuildinToolConfig({
      code: code,
      parameters: GetDefaultToolConfigIfInvalid(code, []),
    });
    setToolDefinition(
      GetDefaultToolDefintion(code, {
        name: '',
        description: '',
        parameters: '',
      }),
    );
  };

  const validateForm = () => {
    if (!toolDefinition.name) {
      setErrorMessage('Please provide a valid name for tool.');
      return false;
    }
    if (!/^[a-zA-Z0-9_]+$/.test(toolDefinition.name)) {
      setErrorMessage(
        'Name should only contain letters, numbers, and underscores.',
      );
      return false;
    }

    if (!toolDefinition.description) {
      setErrorMessage('Please provide a valid description for the tool.');
      return false;
    }
    if (!toolDefinition.parameters) {
      setErrorMessage('Please provide a valid parameters for the tool.');
      return false;
    }
    try {
      JSON.parse(toolDefinition.parameters);
    } catch (error) {
      setErrorMessage(
        'Please provide a valid parameter, parameter must be a valid JSON.',
      );
      return false;
    }

    if (
      !ValidateToolDefaultOptions(
        buildinToolConfig.code,
        buildinToolConfig.parameters,
      )
    ) {
      setErrorMessage('Please provide valid options for action.');
      return false;
    }

    return true;
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage('');
    if (!validateForm()) return;
    if (props.onValidateConfig) {
      const parentError = props.onValidateConfig({
        name: toolDefinition.name,
        description: toolDefinition.description,
        fields: toolDefinition.parameters,
        buildinToolConfig,
      });
      if (parentError) {
        setErrorMessage(parentError);
        return;
      }
    }

    if (props.onChange) {
      props.onChange({
        name: toolDefinition.name,
        description: toolDefinition.description,
        fields: toolDefinition.parameters,
        buildinToolConfig,
      });
    }
  };

  return (
    <GenericModal
      className="flex"
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
    >
      <ModalFitHeightBlock className="w-[1000px]">
        <ModalHeader
          onClose={() => {
            props.setModalOpen(false);
          }}
          title={'Configure Assistant Tool'}
        >
          <ModalTitleBlock>Configure Assistant Tool</ModalTitleBlock>
        </ModalHeader>
        <ModalBody className="overflow-auto max-h-[80dvh] px-0 space-y-0">
          <BuildinTool
            onChangeToolDefinition={setToolDefinition}
            toolDefinition={toolDefinition}
            inputClass={'bg-white'}
            onChangeBuildinTool={onChangeBuildinToolConfig}
            onChangeConfig={setBuildinToolConfig}
            config={buildinToolConfig}
          />
        </ModalBody>
        <ModalFooter errorMessage={errorMessage}>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => {
              props.setModalOpen(false);
            }}
          >
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={onSubmit}
          >
            Save tool
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
};

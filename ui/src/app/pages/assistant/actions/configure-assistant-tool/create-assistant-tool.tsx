import React, { FC, useState } from 'react';
import { CONFIG } from '@/configs';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { useCurrentCredential } from '@/hooks/use-credential';
import {
  BuildinTool,
  BuildinToolConfig,
  GetDefaultToolConfigIfInvalid,
  GetDefaultToolDefintion,
  ValidateToolDefaultOptions,
} from '@/app/components/tools';
import { ToolDefinitionForm } from '@/app/components/tools/common';
import { CreateAssistantTool } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import { TabForm } from '@/app/components/form/tab-form';
import { ButtonSet } from '@carbon/react';

export const CreateTool: FC<{ assistantId: string }> = ({ assistantId }) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const { loading, showLoader, hideLoader } = useRapidaStore();

  const [activeTab, setActiveTab] = useState('action');
  const [errorMessage, setErrorMessage] = useState('');

  const defaultToolCode =
    CONFIG.workspace.features?.knowledge !== false
      ? 'knowledge_retrieval'
      : 'endpoint';

  const [buildinToolConfig, setBuildinToolConfig] = useState<BuildinToolConfig>(
    {
      code: defaultToolCode,
      parameters: GetDefaultToolConfigIfInvalid(defaultToolCode, []),
    },
  );

  const [toolDefinition, setToolDefinition] = useState(
    GetDefaultToolDefintion(defaultToolCode, {
      name: '',
      description: '',
      parameters: '',
    }),
  );

  const onChangeBuildinToolConfig = (code: string) => {
    setActiveTab('action');
    setBuildinToolConfig({
      code: code,
      parameters: GetDefaultToolConfigIfInvalid(
        code,
        buildinToolConfig.parameters,
      ),
    });
    setToolDefinition(
      GetDefaultToolDefintion(code, {
        name: '',
        description: '',
        parameters: '',
      }),
    );
  };

  const isMCP = buildinToolConfig.code === 'mcp';

  const validateAction = (): boolean => {
    setErrorMessage('');
    const err = ValidateToolDefaultOptions(
      buildinToolConfig.code,
      buildinToolConfig.parameters,
    );
    if (err) {
      setErrorMessage(err);
      return false;
    }
    return true;
  };

  const onSubmit = () => {
    setErrorMessage('');
    if (!isMCP) {
      if (!toolDefinition.name) {
        setErrorMessage('Please provide a valid name for tool.');
        return;
      }
      if (!/^[a-zA-Z0-9_]+$/.test(toolDefinition.name)) {
        setErrorMessage(
          'Name should only contain letters, numbers, and underscores.',
        );
        return;
      }
      if (!toolDefinition.parameters) {
        setErrorMessage('Please provide valid parameters for the tool.');
        return;
      }
      try {
        JSON.parse(toolDefinition.parameters);
      } catch {
        setErrorMessage(
          'Please provide valid parameters, must be a valid JSON.',
        );
        return;
      }
    }

    showLoader();
    CreateAssistantTool(
      connectionConfig,
      assistantId,
      toolDefinition.name,
      toolDefinition.description,
      JSON.parse(toolDefinition.parameters),
      buildinToolConfig.code,
      buildinToolConfig.parameters,
      (err, response) => {
        hideLoader();
        if (err) {
          setErrorMessage(
            'Unable to create assistant tool, please check and try again.',
          );
          return;
        }
        if (response?.getSuccess()) {
          toast.success(
            `${response.getData()?.getName()} added to assistant tools successfully`,
          );
          navigator.goToConfigureAssistantTool(assistantId);
        } else {
          if (response?.getError()) {
            const message = response.getError()?.getHumanmessage();
            if (message) {
              setErrorMessage(message);
              return;
            }
          }
          setErrorMessage(
            'Unable to create tool for assistant, please try again.',
          );
        }
      },
      {
        'x-auth-id': authId,
        authorization: token,
        'x-project-id': projectId,
      },
    );
  };

  return (
    <>
      <ConfirmDialogComponent />
      <TabForm
        formHeading="Complete all steps to configure your tool."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'action',
            name: 'Action',
            description:
              'Select the action type and configure its connection options.',
            actions: [
              <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                <SecondaryButton
                  size="lg"
                  onClick={() => showDialog(navigator.goBack)}
                >
                  Cancel
                </SecondaryButton>
                <PrimaryButton
                  size="lg"
                  isLoading={isMCP ? loading : undefined}
                  onClick={() => {
                    if (isMCP) {
                      if (validateAction()) onSubmit();
                    } else {
                      if (validateAction()) setActiveTab('definition');
                    }
                  }}
                >
                  {isMCP ? 'Configure Tool' : 'Continue'}
                </PrimaryButton>
              </ButtonSet>,
            ],
            body: (
              <BuildinTool
                toolDefinition={toolDefinition}
                onChangeToolDefinition={setToolDefinition}
                onChangeBuildinTool={onChangeBuildinToolConfig}
                onChangeConfig={setBuildinToolConfig}
                config={buildinToolConfig}
                showDefinitionForm={false}
              />
            ),
          },
          ...(!isMCP
            ? [
                {
                  code: 'definition',
                  name: 'Definition',
                  description:
                    'Define the tool name, description, and JSON parameter schema.',
                  actions: [
                    <ButtonSet className="!w-full [&>button]:!flex-1 [&>button]:!max-w-none">
                      <SecondaryButton
                        size="lg"
                        onClick={() => showDialog(navigator.goBack)}
                      >
                        Cancel
                      </SecondaryButton>
                      <PrimaryButton
                        size="lg"
                        isLoading={loading}
                        onClick={onSubmit}
                      >
                        Configure Tool
                      </PrimaryButton>
                    </ButtonSet>,
                  ],
                  body: (
                    <ToolDefinitionForm
                      toolDefinition={toolDefinition}
                      onChangeToolDefinition={setToolDefinition}
                      documentationPath="/assistants/tools"
                    />
                  ),
                },
              ]
            : []),
        ]}
      />
    </>
  );
};

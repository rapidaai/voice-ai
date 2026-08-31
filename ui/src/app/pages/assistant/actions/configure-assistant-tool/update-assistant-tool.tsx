import React, { FC, useEffect, useState } from 'react';
import { CONFIG } from '@/configs';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { PrimaryButton, SecondaryButton } from '@/app/components/carbon/button';
import { ButtonSet } from '@carbon/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import {
  BuildinTool,
  BuildinToolConfig,
  GetDefaultToolConfigIfInvalid,
  GetDefaultToolDefintion,
  ValidateToolDefaultOptions,
} from '@/app/components/tools';
import { ToolDefinitionForm } from '@/app/components/tools/common';
import { GetAssistantTool, UpdateAssistantTool } from '@rapidaai/react';
import { useParams } from 'react-router-dom';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import { TabForm } from '@/app/components/form/tab-form';

export const UpdateTool: FC<{ assistantId: string }> = ({ assistantId }) => {
  const navigator = useGlobalNavigation();
  const { assistantToolId } = useParams();
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

  const [toolDefinition, setToolDefinition] = useState<{
    name: string;
    description: string;
    parameters: string;
  }>(
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

  useEffect(() => {
    showLoader();
    GetAssistantTool(
      connectionConfig,
      assistantId,
      assistantToolId!,
      (err, res) => {
        hideLoader();
        if (err) {
          toast.error('Unable to load tool, please try again later.');
          return;
        }
        const wb = res?.getData();
        if (wb) {
          setToolDefinition({
            name: wb.getName(),
            description: wb.getDescription(),
            parameters: JSON.stringify(wb.getFields()?.toJavaScript(), null, 2),
          });
          setBuildinToolConfig({
            code: wb.getExecutionmethod(),
            parameters: GetDefaultToolConfigIfInvalid(
              wb.getExecutionmethod(),
              wb.getExecutionoptionsList(),
            ),
          });
        }
      },
      {
        'x-auth-id': authId,
        authorization: token,
        'x-project-id': projectId,
      },
    );
  }, [assistantId, assistantToolId, authId, token, projectId]);

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
      if (!toolDefinition.description) {
        setErrorMessage('Please provide a description for the tool.');
        return;
      }
      if (!toolDefinition.parameters) {
        setErrorMessage('Please provide valid parameters for the tool.');
        return;
      }
      try {
        JSON.parse(toolDefinition.parameters);
      } catch {
        setErrorMessage('Fields must be a valid JSON.');
        return;
      }
    }

    showLoader();
    UpdateAssistantTool(
      connectionConfig,
      assistantId,
      assistantToolId!,
      toolDefinition.name,
      toolDefinition.description,
      JSON.parse(toolDefinition.parameters),
      buildinToolConfig.code,
      buildinToolConfig.parameters,
      (err, response) => {
        hideLoader();
        if (err) {
          setErrorMessage(
            'Unable to update assistant tool, please check and try again.',
          );
          return;
        }
        if (response?.getSuccess()) {
          toast.success('Assistant tool updated successfully.');
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
            'Unable to update tool for assistant, please try again.',
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
        formHeading="Update all steps to reconfigure your tool."
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
                  {isMCP ? 'Update Tool' : 'Continue'}
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
                        Update Tool
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

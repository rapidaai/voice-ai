import React, { FC, useEffect, useState } from 'react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { useCurrentCredential } from '@/hooks/use-credential';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import {
  BuildinTool,
  BuildinToolConfig,
  GetDefaultToolConfigIfInvalid,
  GetDefaultToolDefintion,
  ValidateToolDefaultOptions,
} from '@/app/components/tools';
import { GetAssistantTool, UpdateAssistantTool } from '@rapidaai/react';
import { useParams } from 'react-router-dom';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';

export const UpdateTool: FC<{ assistantId: string }> = ({ assistantId }) => {
  const navigator = useGlobalNavigation();
  const { assistantToolId } = useParams();
  const { authId, token, projectId } = useCurrentCredential();
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const { loading, showLoader, hideLoader } = useRapidaStore();

  /**
   * buildin tools
   */
  const [buildinToolConfig, setBuildinToolConfig] = useState<BuildinToolConfig>(
    {
      code: 'knowledge_retrieval',
      parameters: GetDefaultToolConfigIfInvalid('knowledge_retrieval', []),
    },
  );

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

  const [toolDefinition, setToolDefinition] = useState<{
    name: string;
    description: string;
    parameters: string;
  }>({
    name: '',
    description: '',
    parameters: '',
  });
  const [errorMessage, setErrorMessage] = useState('');
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
      setErrorMessage('Please provide a description for the tool.');
      return false;
    }
    if (!toolDefinition.parameters) {
      setErrorMessage('Please provide a valid parameters for the tool.');
      return false;
    }
    try {
      JSON.parse(toolDefinition.parameters);
    } catch (error) {
      setErrorMessage('Fields must be a valid JSON.');
      return false;
    }

    if (
      !ValidateToolDefaultOptions(
        buildinToolConfig.code,
        buildinToolConfig.parameters,
      )
    ) {
      setErrorMessage('Please provide valid expected action options.');
      return false;
    }

    return true;
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage('');
    if (!validateForm()) return;

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
        }
        if (response?.getSuccess()) {
          toast.success('Assistant tool updated successfully.');
          navigator.goToConfigureAssistantTool(assistantId);
        } else {
          if (response?.getError()) {
            let err = response.getError();
            const message = err?.getHumanmessage();
            if (message) {
              setErrorMessage(message);
              return;
            }
            setErrorMessage(
              'Unable to update tool for assistant, please check and try again.',
            );
            return;
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

  useEffect(() => {
    // show loading
    showLoader();
    //
    GetAssistantTool(
      connectionConfig,
      assistantId,
      assistantToolId!,
      (err, res) => {
        hideLoader();
        if (err) {
          toast.error('Unable to assistant analysis, please try again later.');
          return;
        }
        // Set state with fetched data
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

  return (
    <form
      onSubmit={onSubmit}
      method="POST"
      className="relative flex flex-col flex-1"
    >
      <ConfirmDialogComponent />
      <div className="overflow-auto flex flex-col flex-1 pb-20 bg-white dark:bg-gray-900">
        <PageHeaderBlock className="border-b">
          <div className="flex items-center gap-3">
            <PageTitleBlock>Update Tool</PageTitleBlock>
          </div>
        </PageHeaderBlock>
        <BuildinTool
          toolDefinition={toolDefinition}
          onChangeToolDefinition={setToolDefinition}
          onChangeBuildinTool={onChangeBuildinToolConfig}
          onChangeConfig={setBuildinToolConfig}
          config={buildinToolConfig}
        />
      </div>

      <PageActionButtonBlock errorMessage={errorMessage}>
        <ICancelButton
          className="px-4 rounded-[2px]"
          onClick={() => showDialog(navigator.goBack)}
          type="button"
        >
          Cancel
        </ICancelButton>
        <IBlueBGButton
          isLoading={loading}
          type="submit"
          className="px-4 rounded-[2px]"
        >
          Update Tool
        </IBlueBGButton>
      </PageActionButtonBlock>
    </form>
  );
};

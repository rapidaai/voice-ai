import React, { FC, useState } from 'react';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { useCurrentCredential } from '@/hooks/use-credential';
import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import {
  BuildinTool,
  BuildinToolConfig,
  GetDefaultToolConfigIfInvalid,
  GetDefaultToolDefintion,
  ValidateToolDefaultOptions,
} from '@/app/components/tools';
import { CreateAssistantTool } from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/hooks';
import { connectionConfig } from '@/configs';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';

export const CreateTool: FC<{ assistantId: string }> = ({ assistantId }) => {
  const navigator = useGlobalNavigation();
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

  const [toolDefinition, setToolDefinition] = useState(
    GetDefaultToolDefintion('knowledge_retrieval', {
      name: '',
      description: '',
      parameters: '',
    }),
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

  const [errorMessage, setErrorMessage] = useState('');
  const validateForm = () => {
    if (!toolDefinition.name) {
      setErrorMessage('Please provide a valid name for tool.');
      return false;
    }
    if (!/^[a-zA-Z0-9_]+$/.test(toolDefinition.name)) {
      setErrorMessage(
        'Please provide valid name, should only contain letters, numbers, and underscores.',
      );
      return false;
    }

    if (!toolDefinition.description) {
      setErrorMessage('Please provide a description for the tool.');
      return false;
    }
    if (!toolDefinition.parameters) {
      setErrorMessage('Please provide valid parameters for the tool.');
      return false;
    }

    try {
      JSON.parse(toolDefinition.parameters);
    } catch (error) {
      setErrorMessage(
        'Please provide a valid parameters, must be a valid JSON.',
      );
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
        }
        if (response?.getSuccess()) {
          toast.success(
            `${response.getData()?.getName()} added to assistant tools successfully`,
          );
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
              'Unable to create tool for assistant, please check and try again.',
            );
            return;
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
    <form
      onSubmit={onSubmit}
      method="POST"
      className="relative flex flex-col flex-1"
    >
      <ConfirmDialogComponent />
      <div className="overflow-auto flex flex-col flex-1 pb-20 bg-white dark:bg-gray-900">
        <PageHeaderBlock className="border-b">
          <div className="flex items-center gap-3">
            <PageTitleBlock>Adding New Tool</PageTitleBlock>
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
        <IBlueBGArrowButton
          isLoading={loading}
          type="submit"
          className="px-4 rounded-[2px]"
        >
          Configure Tool
        </IBlueBGArrowButton>
      </PageActionButtonBlock>
    </form>
  );
};

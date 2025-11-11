import { FC, useEffect, useState } from 'react';
import { useRapidaStore } from '@/hooks';
import { useCredential } from '@/hooks/use-credential';
import { useParams } from 'react-router-dom';
import { Helmet } from '@/app/components/Helmet';
import {
  IBlueBGArrowButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { TabForm } from '@/app/components/Form/tab-form';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { useConfirmDialog } from '@/app/pages/assistant/actions/hooks/use-confirmation';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import {
  AssistantDefinition,
  CreateAssistantProviderRequest,
  GetAssistantProviderResponse,
  GetAssistantRequest,
} from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { Textarea } from '@/app/components/Form/Textarea';
import { ErrorContainer } from '@/app/components/error-container';
import { CreateAssistantProvider, GetAssistant } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { ExternalLink, Info } from 'lucide-react';
import { Input } from '@/app/components/Form/Input';
import { APiHeader } from '@/app/components/external-api/api-header';
import { APiParameter } from '@/app/components/external-api/api-parameter';

export function CreateWebsocketVersion() {
  const { assistantId } = useParams();
  const { goToAssistantListing } = useGlobalNavigation();

  if (!assistantId)
    return (
      <div className="flex flex-1">
        <ErrorContainer
          onAction={goToAssistantListing}
          code="403"
          actionLabel="Go to listing"
          title="Assistant not available"
          description="This assistant may be archived or you don't have access to it. Please check with your administrator or try another assistant."
        />
      </div>
    );

  return <CreateNewVersion assistantId={assistantId!} />;
}
/**
 *
 * @param props
 * @returns
 */
const CreateNewVersion: FC<{ assistantId: string }> = ({ assistantId }) => {
  const [userId, token, projectId] = useCredential();
  const [activeTab, setActiveTab] = useState('change-assistant');
  const navigator = useGlobalNavigation();
  const [errorMessage, setErrorMessage] = useState('');
  const { showDialog, ConfirmDialogComponent } = useConfirmDialog({});
  const currentDate = new Date().toLocaleDateString();
  const [versionMessage, setVersionMessage] = useState(
    `Changed on ${currentDate}`,
  );
  const { loading, showLoader, hideLoader } = useRapidaStore();

  //   websocket url

  const [websocketUrl, setWebscoketUrl] = useState('');
  const [headers, setHeaders] = useState<{ key: string; value: string }[]>([
    { key: '', value: '' },
  ]);
  const [parameters, setParameters] = useState<
    { key: string; value: string }[]
  >([{ key: '', value: '' }]);

  const validateWebsocket = (): boolean => {
    setErrorMessage('');
    setErrorMessage('');
    if (!websocketUrl.trim()) {
      setErrorMessage('Please provide a valid websocket url.');
      return false; // websocketUrl must not be empty or whitespace
    }

    const websocketPattern = /^wss?:\/\/[\w.-]+(:\d+)?(\/.*)?$/i;
    if (!websocketPattern.test(websocketUrl.trim())) {
      setErrorMessage(
        'Please provide a valid WebSocket URL (ws:// or wss://).',
      );
      return false; // websocketUrl must match websocket URL structure
    }
    // Validate headers: key and value must not be empty if the array isn't empty
    for (const header of headers) {
      if (!header.key.trim() || !header.value.trim()) {
        setErrorMessage(
          'Please provide valid values for headers key and value.',
        );
        return false;
      }
    }

    // Validate parameters: key and value must not be empty if the array isn't empty
    for (const param of parameters) {
      if (!param.key.trim() || !param.value.trim()) {
        setErrorMessage(
          'Please provide valid values for parameters key and value.',
        );
        return false;
      }
    }
    return true;
  };

  const createProviderModel = () => {
    setErrorMessage('');
    if (!versionMessage || versionMessage.trim() === '') {
      setErrorMessage('Please provide a valid version description.');
      return;
    }

    showLoader();

    const request = new CreateAssistantProviderRequest();
    const websocket =
      new CreateAssistantProviderRequest.CreateAssistantProviderWebsocket();

    // websocket
    websocket.setWebsocketurl(websocketUrl);

    // adding header
    headers.forEach(p => {
      websocket.getHeadersMap().set(p.key, p.value);
    });
    // connection parameters
    parameters.forEach(p => {
      websocket.getConnectionparametersMap().set(p.key, p.value);
    });

    request.setWebsocket(websocket);
    request.setAssistantid(assistantId);
    request.setDescription(versionMessage);
    CreateAssistantProvider(connectionConfig, request, {
      authorization: token,
      'x-auth-id': userId,
      'x-project-id': projectId,
    })
      .then((car: GetAssistantProviderResponse) => {
        hideLoader();
        if (car?.getSuccess()) {
          navigator.goToAssistantVersions(assistantId);
        } else {
          const errorMessage =
            'Unable to create assistant version. please try again later.';
          const error = car?.getError();
          if (error) {
            setErrorMessage(error.getHumanmessage());
            return;
          }
          setErrorMessage(errorMessage);
          return;
        }
      })
      .catch(err => {
        setErrorMessage(
          'Unable to create assistant version. please try again later.',
        );
      });
  };

  useEffect(() => {
    showLoader();
    const request = new GetAssistantRequest();
    const assistantDef = new AssistantDefinition();
    assistantDef.setAssistantid(assistantId);
    request.setAssistantdefinition(assistantDef);
    GetAssistant(connectionConfig, request, {
      authorization: token,
      'x-auth-id': userId,
      'x-project-id': projectId,
    })
      .then(response => {
        hideLoader();
        if (response?.getSuccess()) {
          const assistantProvider = response
            .getData()
            ?.getAssistantproviderwebsocket();
          if (assistantProvider) {
            setWebscoketUrl(assistantProvider.getUrl());
            const headersArray: { key: string; value: string }[] = [];
            assistantProvider.getHeadersMap().forEach((value, key) => {
              headersArray.push({ key, value });
            });
            setHeaders(headersArray);

            const _parameters: { key: string; value: string }[] = [];
            assistantProvider.getParametersMap().forEach((value, key) => {
              _parameters.push({ key, value });
            });
            setParameters(_parameters);
          }
          return;
        }
        const error = response?.getError();
        const errorMsg = error
          ? error.getHumanmessage()
          : 'Unable to get your assistant. Please try again later.';
        setErrorMessage(errorMsg);
      })
      .catch(err => {
        hideLoader();
        setErrorMessage(
          'Unable to get your assistant. Please try again later.',
        );
      });
  }, [assistantId]);

  return (
    <>
      <ConfirmDialogComponent />
      <Helmet title="Connect new websocket"></Helmet>
      <TabForm
        className="bg-linear-to-r from-white hover:shadow-alternate to-violet-500/5 dark:from-gray-950/30 dark:via-gray-950/10 dark:to-violet-950/20"
        formHeading="Complete all steps to connect your WebSocket."
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'change-assistant',
            name: 'Configure Connection',
            description:
              'Set up a new WebSocket connection to the server where your agent is running.',
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                type="button"
                isLoading={loading}
                onClick={() => {
                  if (validateWebsocket()) {
                    setActiveTab('commit-assistant');
                  }
                }}
                className="px-4 rounded-[2px]"
              >
                Continue
              </IBlueBGArrowButton>,
            ],
            body: (
              <>
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" strokeWidth={1.5} />
                  <div className="ms-3 text-sm font-medium">
                    Please note that new versions of the assistant will not be
                    deployed automatically.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/assistant/create-new-version"
                    className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
                    rel="noreferrer"
                  >
                    Read documentation
                    <ExternalLink
                      className="shrink-0 w-4 h-4 ml-1.5"
                      strokeWidth={1.5}
                    />
                  </a>
                </YellowNoticeBlock>
                <div className="space-y-6 px-8 max-w-4xl">
                  <FieldSet className="relative w-full">
                    <FormLabel>Websocket Endpoint</FormLabel>
                    <Input
                      placeholder="wss://your-agent-server.com/ws"
                      value={websocketUrl}
                      onChange={v => {
                        setWebscoketUrl(v.target.value);
                      }}
                    />
                  </FieldSet>
                  <FieldSet>
                    <FormLabel>Headers</FormLabel>
                    <APiParameter
                      initialValues={headers}
                      setParameterValue={h => {
                        setHeaders(h);
                      }}
                      actionButtonLabel="Add Header"
                      inputClass="bg-white dark:bg-gray-950!"
                    />
                  </FieldSet>
                  <FieldSet>
                    <FormLabel>Connection Parameters</FormLabel>
                    <APiParameter
                      initialValues={parameters}
                      setParameterValue={v => {
                        setParameters(v);
                      }}
                      actionButtonLabel="Add Parameter"
                      inputClass="bg-white dark:bg-gray-950"
                    />
                  </FieldSet>
                </div>
              </>
            ),
          },
          {
            code: 'commit-assistant',
            name: 'Change definition',
            description:
              'Provide a clear description of the changes made in this version to help others understand what has been updated.',
            actions: [
              <ICancelButton
                className="px-4 rounded-[2px]"
                onClick={() => showDialog(navigator.goBack)}
              >
                Cancel
              </ICancelButton>,
              <IBlueBGArrowButton
                isLoading={loading}
                type="button"
                onClick={() => {
                  createProviderModel();
                }}
                className="px-4 rounded-[2px]"
              >
                Create new version
              </IBlueBGArrowButton>,
            ],
            body: (
              <>
                <YellowNoticeBlock className="flex items-center">
                  <Info className="shrink-0 w-4 h-4" strokeWidth={1.5} />
                  <div className="ms-3 text-sm font-medium">
                    Please note that new versions of the assistant will not be
                    deployed automatically.
                  </div>
                  <a
                    target="_blank"
                    href="https://doc.rapida.ai/assistant/create-new-version"
                    className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
                    rel="noreferrer"
                  >
                    Read documentation
                    <ExternalLink
                      className="shrink-0 w-4 h-4 ml-1.5"
                      strokeWidth={1.5}
                    />
                  </a>
                </YellowNoticeBlock>
                <div className="space-y-6 px-8 max-w-4xl">
                  <FieldSet>
                    <FormLabel>Change description</FormLabel>
                    <Textarea
                      row={5}
                      value={versionMessage}
                      placeholder={'Describe the changes made in this version'}
                      onChange={t => setVersionMessage(t.target.value)}
                    />
                  </FieldSet>
                </div>
              </>
            ),
          },
        ]}
      />
    </>
  );
};

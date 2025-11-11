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
  CreateAssistantProvider,
  CreateAssistantProviderRequest,
  GetAssistantProviderResponse,
  GetAssistantRequest,
} from '@rapidaai/react';
import { FormLabel } from '@/app/components/form-label';
import { Textarea } from '@/app/components/Form/Textarea';
import { ErrorContainer } from '@/app/components/error-container';
import { GetAssistant } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { ExternalLink, Info } from 'lucide-react';
import { Input } from '@/app/components/Form/Input';
import { APiParameter } from '@/app/components/external-api/api-parameter';
import { InputHelper } from '@/app/components/input-helper';
import { CodeEditor } from '@/app/components/Form/editor/code-editor';
import toast from 'react-hot-toast/headless';

export function CreateAgentKitVersion() {
  /**
   * get all the models when type change
   */
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

  const [agentKitUrl, setAgentKitUrl] = useState('');
  const [certificate, setCertificate] = useState('');
  const [parameters, setParameters] = useState<
    {
      key: string;
      value: string;
    }[]
  >([]);
  const validateAgentKit = (): boolean => {
    const grpcUrlPattern = /^[a-zA-Z0-9.-]+(:\d+)?$/; // Matches "hostname" or "hostname:port"
    const sslCertPattern =
      /^-----BEGIN CERTIFICATE-----[\s\S]+-----END CERTIFICATE-----$/;

    if (!grpcUrlPattern.test(agentKitUrl)) {
      setErrorMessage(
        'Invalid gRPC URL. It should follow the format "hostname" or "hostname:port".',
      );
      return false;
    }

    if (certificate && !sslCertPattern.test(certificate)) {
      setErrorMessage(
        'Invalid SSL certificate format. It should start with "-----BEGIN CERTIFICATE-----" and end with "-----END CERTIFICATE-----".',
      );
      return false;
    }

    const hasInvalidParameter = parameters.some(
      param => !param.key.trim() || !param.value.trim(),
    );
    if (hasInvalidParameter) {
      setErrorMessage('All parameters must have non-empty keys and values.');
      return false;
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
    const agentKit =
      new CreateAssistantProviderRequest.CreateAssistantProviderAgentkit();

    agentKit.setAgentkiturl(agentKitUrl);
    agentKit.setCertificate(certificate);
    parameters.forEach(p => {
      agentKit.getMetadataMap().set(p.key, p.value);
    });

    //
    request.setAgentkit(agentKit);
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
          toast.success(
            'Assistant version with model has been created successfully.',
          );
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
            ?.getAssistantprovideragentkit();
          if (assistantProvider) {
            setAgentKitUrl(assistantProvider.getUrl());
            setCertificate(assistantProvider.getCertificate());
            const _parameters: { key: string; value: string }[] = [];
            assistantProvider.getMetadataMap().forEach((value, key) => {
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
      <Helmet title="Connect AgentKit"></Helmet>
      <TabForm
        formHeading="Complete all steps to connect AgentKit."
        className="bg-linear-to-r from-white hover:shadow-alternate to-violet-500/5 dark:from-gray-950/30 dark:via-gray-950/10 dark:to-violet-950/20"
        activeTab={activeTab}
        onChangeActiveTab={() => {}}
        errorMessage={errorMessage}
        form={[
          {
            code: 'change-assistant',
            name: 'Connect configuration',
            description:
              'Provide the connection configuration for your Rapida AgentKit setup.',
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
                  if (validateAgentKit()) {
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
                    <FormLabel>AgentKit Endpoint</FormLabel>
                    <Input
                      placeholder="agent.your-domain.com:5051"
                      value={agentKitUrl}
                      onChange={v => {
                        setAgentKitUrl(v.target.value);
                      }}
                    />
                    <InputHelper>
                      The gRPC server address where your Rapida AgentKit is
                      running.
                    </InputHelper>
                  </FieldSet>
                  <FieldSet>
                    <FormLabel>TLS Certificate (Optional)</FormLabel>
                    <CodeEditor
                      placeholder="-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----"
                      value={certificate}
                      onChange={value => {
                        setCertificate(certificate);
                      }}
                      className="min-h-40 max-h-dvh "
                    />
                    <InputHelper>
                      Custom CA certificate for server verification (optional,
                      leave empty for system defaults)
                    </InputHelper>
                  </FieldSet>
                  <FieldSet>
                    <FormLabel>Metadata</FormLabel>
                    <APiParameter
                      actionButtonLabel="Add Metadata"
                      setParameterValue={parameters => {
                        setParameters(parameters);
                      }}
                      initialValues={parameters}
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
              'Provide a change definition that helps people understand what has changed in this version.',
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

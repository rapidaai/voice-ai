import { useState, useContext, useCallback, useEffect, FC } from 'react';
import {
  CreateProjectCredential,
  GetAllProjectCredential,
} from '@rapidaai/react';
import { useCredential, useCurrentCredential } from '@/hooks/use-credential';
import {
  CreateProjectCredentialResponse,
  GetAllProjectCredentialResponse,
  ProjectCredential,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/Helmet';
import { ActionableEmptyMessage } from '@/app/components/container/message/actionable-empty-message';
import { AuthContext } from '@/context/auth-context';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import { ExternalLink, Info, Plus, RotateCw } from 'lucide-react';
import { connectionConfig } from '@/configs';
import { Eye, EyeOff, Copy, CheckCircle } from 'lucide-react';
import { toHumanReadableDate } from '@/styles/media';
import { Card, CardTitle } from '@/app/components/base/cards';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { FormLabel } from '@/app/components/form-label';
import { CopyButton } from '@/app/components/Form/Button/copy-button';
/**
 *
 * @returns
 */
export function ProjectCredentialPage() {
  /**
   * all the result
   */
  const [ourKeys, setOurKeys] = useState<ProjectCredential[]>([]);

  /**
   * Current project credential
   */
  const { currentProjectRole } = useContext(AuthContext);

  /**
   * authentication
   */
  const [userId, token] = useCredential();

  /**
   * on create project credential
   */
  const onCreateProjectCredential = () => {
    if (!currentProjectRole) return;
    CreateProjectCredential(
      connectionConfig,
      currentProjectRole?.projectid,
      'publishable key',
      afterCreateProjectCredential,
      {
        authorization: token,
        'x-auth-id': userId,
      },
    );
  };

  /**
   * after create project credential
   */
  const afterCreateProjectCredential = useCallback(
    (err, data: CreateProjectCredentialResponse | null) => {
      if (data?.getSuccess()) {
        getAllProjectCredential();
      } else {
        let errorMessage = data?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
        } else {
          toast.error(
            'Unable to process your request. please try again later.',
          );
        }
      }
    },
    [],
  );

  /**
   * after get all the project credentials
   */
  const afterGetAllProjectCredential = useCallback(
    (err, data: GetAllProjectCredentialResponse | null) => {
      if (data?.getSuccess()) {
        setOurKeys(data.getDataList());
      } else {
        let errorMessage = data?.getError();
        if (errorMessage) {
          toast.error(errorMessage.getHumanmessage());
        } else {
          toast.error(
            'Unable to process your request. please try again later.',
          );
        }
      }
    },
    [],
  );
  //   getting all the publishable keys
  // load all the project credentials call it publishable key
  useEffect(() => {
    getAllProjectCredential();
  }, [currentProjectRole]);

  //   when someone add things then reload the state
  const shouldReload = () => {
    getAllProjectCredential();
  };

  const getAllProjectCredential = () => {
    if (currentProjectRole)
      GetAllProjectCredential(
        connectionConfig,
        currentProjectRole.projectid,
        afterGetAllProjectCredential,
        {
          authorization: token,
          'x-auth-id': userId,
        },
      );
  };
  /**
   *
   */
  return (
    <>
      <Helmet title="Providers and Models" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Project Developer Keys</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${ourKeys.length}/${ourKeys.length}`}
          </div>
        </div>
        <div className="flex divide-x">
          <IButton
            className="border-r"
            onClick={() => {
              shouldReload();
            }}
          >
            Reload keys
            <RotateCw className="w-4 h-4 ml-1.5" strokeWidth={1.5} />
          </IButton>
          <IBlueButton
            onClick={() => {
              onCreateProjectCredential();
            }}
          >
            Create new credential
            <Plus className="w-4 h-4 ml-1.5" />
          </IBlueButton>
        </div>
      </PageHeaderBlock>
      <YellowNoticeBlock className="flex items-center">
        <Info className="shrink-0 w-4 h-4" />
        <div className="ms-3 text-sm font-medium">
          These are project-specific keys. They are used to authenticate and
          interact with the Rapida service for this particular project.
        </div>
        <a
          target="_blank"
          href="https://doc.rapida.ai/integrations/rapida-credentials"
          className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
          rel="noreferrer"
        >
          Read documentation
          <ExternalLink className="shrink-0 w-4 h-4 ml-1.5" strokeWidth={1.5} />
        </a>
      </YellowNoticeBlock>
      {ourKeys && ourKeys.length > 0 ? (
        <div className="grid grid-cols-3 gap-3 px-4 py-4 flex-1 overflow-auto">
          {ourKeys.map((pc, idx) => {
            return (
              <div key={idx}>
                <CredentialCard credential={pc}></CredentialCard>
              </div>
            );
          })}
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center">
          <ActionableEmptyMessage
            title="No credentials"
            subtitle="There are no SDK Authentication Credential found to display"
            action="Create new credential"
            onActionClick={() => {
              onCreateProjectCredential();
            }}
          />
        </div>
      )}
    </>
  );
}

// CredentialCard Component - Contains all card-specific logic
const CredentialCard: FC<{ credential: ProjectCredential }> = ({
  credential,
}) => {
  const [isVisible, setIsVisible] = useState(false);
  const [isCopied, setIsCopied] = useState(false);

  const copyToClipboard = async text => {
    try {
      await navigator.clipboard.writeText(text);
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const maskCredential = credential => {
    return credential.substring(0, 6) + '•'.repeat(36);
  };

  return (
    <Card>
      <CardTitle>
        <div className="flex items-center gap-2 justify-between">
          <h3 className="font-semibold  truncate">{credential.getName()}</h3>
          <div className="flex items-center gap-1 text-sm">
            {toHumanReadableDate(credential.getCreateddate()!)}
          </div>
        </div>
      </CardTitle>

      <div className="border-t -mx-4 mt-4 p-4">
        <div className="flex items-center gap-2">
          <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
            {isVisible
              ? credential.getKey()
              : maskCredential(credential.getKey())}
          </code>

          <div className="flex shrink-0 border divide-x">
            <IButton
              onClick={() => setIsVisible(!isVisible)}
              title={isVisible ? 'Hide' : 'Show'}
            >
              {isVisible ? (
                <EyeOff className="w-4 h-4 " />
              ) : (
                <Eye className="w-4 h-4 " />
              )}
            </IButton>

            <IButton
              onClick={() => copyToClipboard(credential.getKey())}
              title="Copy"
            >
              {isCopied ? (
                <CheckCircle className="w-4 h-4 text-emerald-400" />
              ) : (
                <Copy className="w-4 h-4 " />
              )}
            </IButton>
          </div>
        </div>
      </div>
    </Card>
  );
};

export function PersonalCredentialPage() {
  /**
   * authentication
   */
  const { token, authId, projectId } = useCurrentCredential();

  const [isVisible, setIsVisible] = useState(false);
  const [isCopied, setIsCopied] = useState(false);

  const copyToClipboard = async text => {
    try {
      await navigator.clipboard.writeText(text);
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  const maskCredential = credential => {
    return credential.substring(0, 6) + '•'.repeat(36);
  };

  /**
   *
   */
  return (
    <>
      <Helmet title="Providers and Models" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Personal Tokens</PageTitleBlock>
        </div>
      </PageHeaderBlock>
      <YellowNoticeBlock>
        These are your personal access tokens. They are used to authenticate and
        interact with the Rapida service across all your projects.
        <a
          href="https://doc.rapida.ai/integrations/rapida-credentials"
          className="mx-2 hover:underline font-semibold"
        >
          Learn more in the documentation
        </a>
      </YellowNoticeBlock>
      <div className="grid grid-cols-3 gap-3 px-4 py-4 overflow-auto">
        <Card className="h-auto flex-none!">
          <CardTitle>
            <div className="flex items-center gap-2 justify-between">
              <h3 className="font-semibold  truncate">Personal Token</h3>
            </div>
          </CardTitle>

          <div className="border-t -mx-4 mt-4 p-4 space-y-6">
            <FieldSet>
              <FormLabel>Authorization</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
                  {isVisible ? token : maskCredential(token)}
                </code>

                <div className="flex shrink-0 border divide-x">
                  <IButton
                    onClick={() => setIsVisible(!isVisible)}
                    title={isVisible ? 'Hide' : 'Show'}
                  >
                    {isVisible ? (
                      <EyeOff className="w-4 h-4 " />
                    ) : (
                      <Eye className="w-4 h-4 " />
                    )}
                  </IButton>

                  <IButton onClick={() => copyToClipboard(token)} title="Copy">
                    {isCopied ? (
                      <CheckCircle className="w-4 h-4 text-emerald-400" />
                    ) : (
                      <Copy className="w-4 h-4 " />
                    )}
                  </IButton>
                </div>
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>x-auth-id</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
                  {isVisible ? authId : maskCredential(authId)}
                </code>

                <div className="flex shrink-0 border divide-x">
                  <CopyButton className="h-8 w-8">{authId}</CopyButton>
                </div>
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>Project ID</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 dark:bg-gray-900 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden">
                  {isVisible ? projectId : maskCredential(projectId)}
                </code>
                <div className="flex shrink-0 border divide-x">
                  <CopyButton className="h-8 w-8">{projectId}</CopyButton>
                </div>
              </div>
            </FieldSet>
          </div>
        </Card>
      </div>
    </>
  );
}

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
import { Helmet } from '@/app/components/helmet';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { AuthContext } from '@/context/auth-context';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { PageTitleWithCount } from '@/app/components/blocks/page-title-with-count';
import {
  Add,
  Renew,
  View,
  ViewOff,
  Copy,
  Checkmark,
} from '@carbon/icons-react';
import { connectionConfig } from '@/configs';
import { toHumanReadableDate } from '@/utils/date';
import { DocNoticeBlock } from '@/app/components/container/message/notice-block/doc-notice-block';
import { FieldSet } from '@/app/components/form/fieldset';
import { FormLabel } from '@/app/components/form-label';
import { CopyButton } from '@/app/components/carbon/button/copy-button';
import { GhostButton } from '@/app/components/carbon/button';
import { BaseCard } from '@/app/components/base/cards';

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
      <PageHeaderBlock className="border-b">
        <PageTitleWithCount count={ourKeys.length} total={ourKeys.length}>
          Project Developer Keys
        </PageTitleWithCount>
        <div className="flex items-stretch h-12 border-l border-gray-200 dark:border-gray-800">
          <GhostButton size="md" onClick={shouldReload} className="h-full">
            <Renew size={16} />
          </GhostButton>
          <button
            type="button"
            onClick={onCreateProjectCredential}
            className="flex items-center gap-2 px-4 text-sm text-white bg-primary hover:bg-primary/90 transition-colors whitespace-nowrap"
          >
            Create credential
            <Add size={16} />
          </button>
        </div>
      </PageHeaderBlock>
      <DocNoticeBlock docPath="/integrations/rapida-credentials">
        These are project-specific keys. They are used to authenticate and
        interact with the platform service for this particular project.
      </DocNoticeBlock>
      {ourKeys && ourKeys.length > 0 ? (
        <section className="grid content-start grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 grow shrink-0 m-4">
          {ourKeys.map((pc, idx) => (
            <CredentialCard key={idx} credential={pc} />
          ))}
        </section>
      ) : (
        <div className="flex-1 flex items-center justify-center">
          <EmptyState
            title="No credentials"
            subtitle="There are no SDK Authentication Credential found to display"
            action="Create new credential"
            onAction={onCreateProjectCredential}
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
    <BaseCard>
      {/* Card header */}
      <div className="px-4 pt-4 pb-3">
        <div className="flex items-center justify-between gap-2">
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
            {credential.getName()}
          </span>
          <span className="text-xs tabular-nums text-gray-500 dark:text-gray-400 shrink-0">
            {toHumanReadableDate(credential.getCreateddate()!)}
          </span>
        </div>
      </div>

      {/* Key row */}
      <div className="px-4 pb-4 border-t border-gray-100 dark:border-gray-800 pt-3">
        <div className="flex items-center gap-2">
          <code className="flex-1 bg-gray-50 dark:bg-gray-950 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
            {isVisible
              ? credential.getKey()
              : maskCredential(credential.getKey())}
          </code>
          <div className="flex shrink-0 border border-gray-200 dark:border-gray-700 divide-x divide-gray-200 dark:divide-gray-700">
            <GhostButton
              size="md"
              onClick={() => setIsVisible(!isVisible)}
              title={isVisible ? 'Hide' : 'Show'}
            >
              {isVisible ? <ViewOff size={16} /> : <View size={16} />}
            </GhostButton>
            <GhostButton
              size="md"
              onClick={() => copyToClipboard(credential.getKey())}
              title="Copy"
            >
              {isCopied ? <Checkmark size={16} /> : <Copy size={16} />}
            </GhostButton>
          </div>
        </div>
      </div>
    </BaseCard>
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
      <PageHeaderBlock className="border-b">
        <div className="flex items-center gap-3">
          <PageTitleBlock>Personal Tokens</PageTitleBlock>
        </div>
      </PageHeaderBlock>
      <DocNoticeBlock docPath="/integrations/rapida-credentials">
        These are your personal access tokens. They are used to authenticate and
        interact with the platform service across all your projects.
      </DocNoticeBlock>
      <section className="grid content-start grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 grow shrink-0 m-4">
        <BaseCard>
          {/* Card header */}
          <div className="px-4 pt-4 pb-3">
            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
              Personal Token
            </span>
          </div>

          {/* Fields */}
          <div className="px-4 pb-4 border-t border-gray-100 dark:border-gray-800 pt-3 space-y-4">
            <FieldSet>
              <FormLabel>Authorization</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-gray-50 dark:bg-gray-950 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
                  {isVisible ? token : maskCredential(token)}
                </code>
                <div className="flex shrink-0 border border-gray-200 dark:border-gray-700 divide-x divide-gray-200 dark:divide-gray-700">
                  <GhostButton
                    size="md"
                    onClick={() => setIsVisible(!isVisible)}
                    title={isVisible ? 'Hide' : 'Show'}
                  >
                    {isVisible ? <ViewOff size={16} /> : <View size={16} />}
                  </GhostButton>
                  <GhostButton
                    size="md"
                    onClick={() => copyToClipboard(token)}
                    title="Copy"
                  >
                    {isCopied ? <Checkmark size={16} /> : <Copy size={16} />}
                  </GhostButton>
                </div>
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>x-auth-id</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-gray-50 dark:bg-gray-950 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
                  {isVisible ? authId : maskCredential(authId)}
                </code>
                <div className="flex shrink-0 border border-gray-200 dark:border-gray-700 divide-x divide-gray-200 dark:divide-gray-700">
                  <CopyButton className="h-8 w-8">{authId}</CopyButton>
                </div>
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>Project ID</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-gray-50 dark:bg-gray-950 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
                  {isVisible ? projectId : maskCredential(projectId)}
                </code>
                <div className="flex shrink-0 border border-gray-200 dark:border-gray-700 divide-x divide-gray-200 dark:divide-gray-700">
                  <CopyButton className="h-8 w-8">{projectId}</CopyButton>
                </div>
              </div>
            </FieldSet>
          </div>
        </BaseCard>
      </section>
    </>
  );
}

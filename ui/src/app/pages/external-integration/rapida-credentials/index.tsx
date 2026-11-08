import { useState, useContext, useCallback, useEffect, FC } from 'react';
import { useCredential, useCurrentCredential } from '@/hooks/use-credential';
import {
  CreateProjectCredentialResponse,
  GetAllProjectCredentialResponse,
  ProjectCredential,
} from '@rapidaai/react';
import toast from 'react-hot-toast/headless';
import { Helmet } from '@/app/components/app-shell/helmet';
import { EmptyState } from '@/app/components/ui/feedback';
import { AuthContext } from '@/context/auth-context';
import { PageHeader } from '@/app/components/layout/page-header';
import {
  Add,
  Renew,
  View,
  ViewOff,
  Copy,
  Checkmark,
} from '@carbon/icons-react';
import { toHumanReadableDate } from '@/utils/date';
import { DocNoticeBlock } from '@/app/components/layout/container/message/notice-block/doc-notice-block';
import { FieldSet } from '@/app/components/ui/primitives';
import { FormLabel } from '@/app/components/ui/primitives';
import { CopyButton } from '@/app/components/ui/primitives';
import { GhostButton } from '@/app/components/ui/primitives';
import { IconOnlyButton } from '@/app/components/ui/primitives';
import { BaseCard } from '@/app/components/ui/primitives';
import { PrimaryButton } from '@/app/components/ui/primitives';
import { Text } from '@/app/components/ui/primitives';
import {
  createProjectPublishableCredential,
  listProjectCredentials,
} from '@/clients';

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
    createProjectPublishableCredential({
      projectId: currentProjectRole.projectid,
      name: 'publishable key',
      auth: { token, userId },
      callback: afterCreateProjectCredential,
    });
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
      listProjectCredentials({
        projectId: currentProjectRole.projectid,
        auth: { token, userId },
        callback: afterGetAllProjectCredential,
      });
  };
  /**
   *
   */
  return (
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
    >
      <Helmet title="Providers and Models" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Integrations
          </Text>
        }
        title="Project Developer Keys"
        description="Create and manage project-scoped keys used to authenticate platform service requests."
        contextualActions={
          <Text variant="label-01" className="text-muted tabular-nums">
            {ourKeys.length}
          </Text>
        }
        pageActions={
          <div className="flex w-full items-stretch md:w-auto">
            <IconOnlyButton
              kind="ghost"
              size="md"
              renderIcon={Renew}
              iconDescription="Refresh credentials"
              onClick={shouldReload}
              className="h-full"
            />
            <PrimaryButton
              size="md"
              type="button"
              onClick={onCreateProjectCredential}
              className="h-full whitespace-nowrap"
              renderIcon={Add}
            >
              Create credential
            </PrimaryButton>
          </div>
        }
      />
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
    </div>
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
          <span className="text-sm font-medium text-foreground truncate">
            {credential.getName()}
          </span>
          <span className="text-xs tabular-nums text-muted shrink-0">
            {toHumanReadableDate(credential.getCreateddate()!)}
          </span>
        </div>
      </div>

      {/* Key row */}
      <div className="px-4 pb-4 border-t border-border-subtle pt-3">
        <div className="flex items-center gap-2">
          <code className="flex-1 bg-layer px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
            {isVisible
              ? credential.getKey()
              : maskCredential(credential.getKey())}
          </code>
          <div className="flex shrink-0 border border-border-subtle divide-x divide-border-subtle ">
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
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden"
    >
      <Helmet title="Providers and Models" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Integrations
          </Text>
        }
        title="Personal Tokens"
        description="Use your personal access token to authenticate platform service requests across projects."
      />
      <DocNoticeBlock docPath="/integrations/rapida-credentials">
        These are your personal access tokens. They are used to authenticate and
        interact with the platform service across all your projects.
      </DocNoticeBlock>
      <section className="grid content-start grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 grow shrink-0 m-4">
        <BaseCard>
          {/* Card header */}
          <div className="px-4 pt-4 pb-3">
            <span className="text-sm font-medium text-foreground ">
              Personal Token
            </span>
          </div>

          {/* Fields */}
          <div className="px-4 pb-4 border-t border-border-subtle pt-3 space-y-4">
            <FieldSet>
              <FormLabel>Authorization</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-layer px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
                  {isVisible ? token : maskCredential(token)}
                </code>
                <div className="flex shrink-0 border border-border-subtle divide-x divide-border-subtle ">
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
                <code className="flex-1 bg-layer px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
                  {isVisible ? authId : maskCredential(authId)}
                </code>
                <div className="flex shrink-0 border border-border-subtle divide-x divide-border-subtle ">
                  <CopyButton className="h-8 w-8">{authId}</CopyButton>
                </div>
              </div>
            </FieldSet>
            <FieldSet>
              <FormLabel>Project ID</FormLabel>
              <div className="flex items-center gap-2">
                <code className="flex-1 bg-layer px-3 py-2 font-mono text-xs min-w-0 overflow-hidden truncate">
                  {isVisible ? projectId : maskCredential(projectId)}
                </code>
                <div className="flex shrink-0 border border-border-subtle divide-x divide-border-subtle ">
                  <CopyButton className="h-8 w-8">{projectId}</CopyButton>
                </div>
              </div>
            </FieldSet>
          </div>
        </BaseCard>
      </section>
    </div>
  );
}

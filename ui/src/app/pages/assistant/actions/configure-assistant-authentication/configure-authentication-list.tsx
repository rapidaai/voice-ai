import { FC, useEffect, useMemo, useState } from 'react';
import {
  AssistantConfiguration,
  DeleteAssistantConfiguration,
  DeleteAssistantConfigurationRequest,
  GetAllAssistantConfiguration,
  GetAllAssistantConfigurationRequest,
  Paginate,
  UpdateAssistantConfiguration,
  UpdateAssistantConfigurationRequest,
} from '@rapidaai/react';
import {
  Breadcrumb,
  Button as CarbonButton,
  ModalBody,
  ModalFooter,
  ModalHeader,
  ComposedModal,
  BreadcrumbItem,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  OverflowMenu,
  OverflowMenuItem,
} from '@carbon/react';
import { Renew, Add } from '@carbon/icons-react';
import toast from 'react-hot-toast/headless';

import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { connectionConfig } from '@/configs';
import { PrimaryButton, IconOnlyButton } from '@/app/components/carbon/button';
import { UrlTableCell } from '@/app/components/carbon/url-table-cell';
import { SectionLoader } from '@/app/components/loader/section-loader';
import { TableSection } from '@/app/components/sections/table-section';
import { EmptyState } from '@/app/components/carbon/empty-state';
import { RecordStatusIndicator } from '@/app/components/carbon/record-status-indicator';
import { toHumanReadableDateTime } from '@/utils/date';

import {
  toOptionMap,
  AUTH_OPTION_ENDPOINT,
  AUTH_OPTION_METHOD,
} from './shared';

const authenticationConfigurationType = 'authentication';

type AuthenticationAction = 'enable' | 'disable' | 'delete';

interface ConfigureAuthenticationListProps {
  assistantId: string;
}

export const ConfigureAuthenticationList: FC<
  ConfigureAuthenticationListProps
> = ({ assistantId }) => {
  const navigator = useGlobalNavigation();
  const { authId, token, projectId } = useCurrentCredential();

  const [loading, setLoading] = useState(true);
  const [authentication, setAuthentication] =
    useState<AssistantConfiguration | null>(null);
  const [pendingAction, setPendingAction] =
    useState<AuthenticationAction | null>(null);
  const [submittingAction, setSubmittingAction] = useState(false);

  const configureRoute = `/deployment/assistant/${assistantId}/configure-authentication/${
    authentication ? 'edit' : 'create'
  }`;

  const load = () => {
    setLoading(true);
    const request = new GetAllAssistantConfigurationRequest();
    request.setAssistantid(assistantId);
    request.setConfigurationtype(authenticationConfigurationType);

    const paginate = new Paginate();
    paginate.setPage(1);
    paginate.setPagesize(1);
    request.setPaginate(paginate);

    GetAllAssistantConfiguration(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        if (!response?.getSuccess()) {
          setAuthentication(null);
          setLoading(false);
          return;
        }

        setAuthentication(response.getDataList()?.[0] || null);
        setLoading(false);
      })
      .catch(() => {
        setAuthentication(null);
        setLoading(false);
      });
  };

  useEffect(() => {
    load();
  }, [assistantId, authId, token, projectId]);

  const optionMap = useMemo(
    () => toOptionMap(authentication?.getOptionsList?.() || []),
    [authentication],
  );
  const authenticationEnabled = authentication?.getEnabled?.() ?? true;

  const closeActionModal = () => {
    if (submittingAction) return;
    setPendingAction(null);
  };

  const onDelete = () => {
    if (!authentication) return;
    const request = new DeleteAssistantConfigurationRequest();
    request.setAssistantid(assistantId);
    request.setId(authentication.getId());

    setSubmittingAction(true);
    DeleteAssistantConfiguration(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        if (response?.getSuccess()) {
          toast.success('Assistant authentication deleted successfully.');
          setPendingAction(null);
          load();
          return;
        }
        toast.error(
          response?.getError?.()?.getHumanmessage?.() ||
            'Unable to delete assistant authentication.',
        );
      })
      .catch(err => {
        toast.error(
          err?.message || 'Unable to delete assistant authentication.',
        );
      })
      .finally(() => setSubmittingAction(false));
  };

  const setAuthenticationEnabled = (enabled: boolean) => {
    if (!authentication) return;

    const request = new UpdateAssistantConfigurationRequest();
    request.setId(authentication.getId());
    request.setAssistantid(assistantId);
    request.setConfigurationtype(authenticationConfigurationType);
    request.setProvider(authentication.getProvider() || 'http');
    request.setEnabled(enabled);
    request.setOptionsList(authentication.getOptionsList?.() || []);

    setSubmittingAction(true);
    UpdateAssistantConfiguration(connectionConfig, request, {
      'x-auth-id': authId,
      authorization: token,
      'x-project-id': projectId,
    })
      .then(response => {
        if (response?.getSuccess()) {
          toast.success(
            `Assistant authentication ${enabled ? 'enabled' : 'disabled'} successfully.`,
          );
          setPendingAction(null);
          load();
          return;
        }
        toast.error(
          response?.getError?.()?.getHumanmessage?.() ||
            `Unable to ${enabled ? 'enable' : 'disable'} assistant authentication.`,
        );
      })
      .catch(err => {
        toast.error(
          err?.message ||
            `Unable to ${enabled ? 'enable' : 'disable'} assistant authentication.`,
        );
      })
      .finally(() => setSubmittingAction(false));
  };

  const confirmAction = () => {
    switch (pendingAction) {
      case 'enable':
        setAuthenticationEnabled(true);
        break;
      case 'disable':
        setAuthenticationEnabled(false);
        break;
      case 'delete':
        onDelete();
        break;
      default:
        break;
    }
  };

  const modalTitle =
    pendingAction === 'enable'
      ? 'Enable authentication?'
      : pendingAction === 'disable'
        ? 'Disable authentication?'
        : 'Delete authentication?';
  const modalContent =
    pendingAction === 'enable'
      ? 'Sessions will be verified before initialization.'
      : pendingAction === 'disable'
        ? 'Sessions will continue without authentication verification.'
        : 'Authentication configuration will be removed from this assistant.';
  const modalPrimaryLabel =
    pendingAction === 'enable'
      ? 'Enable'
      : pendingAction === 'disable'
        ? 'Disable'
        : 'Delete';

  if (loading) {
    return (
      <div className="h-full w-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col flex-1">
      <ComposedModal
        open={Boolean(pendingAction)}
        onClose={closeActionModal}
        size="sm"
        danger={pendingAction !== 'enable'}
      >
        <ModalHeader title={modalTitle} />
        <ModalBody>
          <p>{modalContent}</p>
        </ModalBody>
        <ModalFooter danger={pendingAction !== 'enable'}>
          <CarbonButton
            kind="secondary"
            size="md"
            disabled={submittingAction}
            onClick={closeActionModal}
          >
            Cancel
          </CarbonButton>
          <CarbonButton
            kind={pendingAction === 'enable' ? 'primary' : 'danger'}
            size="md"
            disabled={submittingAction}
            onClick={confirmAction}
          >
            {modalPrimaryLabel}
          </CarbonButton>
        </ModalFooter>
      </ComposedModal>
      <div className="px-4 pt-4 pb-6 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div>
          <Breadcrumb noTrailingSlash className="mb-2">
            <BreadcrumbItem
              href={`/deployment/assistant/${assistantId}/overview`}
            >
              Assistant
            </BreadcrumbItem>
          </Breadcrumb>
          <h1 className="text-2xl font-light tracking-tight">Authentication</h1>
        </div>
      </div>

      <TableToolbar>
        <TableToolbarContent>
          <IconOnlyButton
            kind="ghost"
            size="lg"
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={load}
          />
          <PrimaryButton
            size="md"
            renderIcon={Add}
            onClick={() => navigator.goTo(configureRoute)}
          >
            Configure authentication
          </PrimaryButton>
        </TableToolbarContent>
      </TableToolbar>

      <TableSection>
        {authentication ? (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader>Provider Type</TableHeader>
                <TableHeader>Method</TableHeader>
                <TableHeader>URL</TableHeader>
                <TableHeader>Status</TableHeader>
                <TableHeader>Date</TableHeader>
                <TableHeader>Actions</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell className="text-sm whitespace-nowrap">
                  {authentication.getProvider() || 'http'}
                </TableCell>
                <TableCell className="text-sm whitespace-nowrap">
                  {optionMap[AUTH_OPTION_METHOD] || '-'}
                </TableCell>
                <UrlTableCell url={optionMap[AUTH_OPTION_ENDPOINT]} />
                <TableCell className="text-sm whitespace-nowrap">
                  <RecordStatusIndicator
                    state={authentication.getStatus()}
                    textSize={14}
                  />
                </TableCell>
                <TableCell className="text-[13px] whitespace-nowrap">
                  {authentication.getCreateddate() &&
                    toHumanReadableDateTime(authentication.getCreateddate()!)}
                </TableCell>
                <TableCell
                  className="text-sm whitespace-nowrap"
                  onClick={e => e.stopPropagation()}
                >
                  <OverflowMenu
                    size="sm"
                    flipped
                    aria-label="Authentication actions"
                  >
                    <OverflowMenuItem
                      itemText="Edit"
                      onClick={() => navigator.goTo(configureRoute)}
                    />
                    <OverflowMenuItem
                      itemText={authenticationEnabled ? 'Disable' : 'Enable'}
                      onClick={() =>
                        setPendingAction(
                          authenticationEnabled ? 'disable' : 'enable',
                        )
                      }
                    />
                    <OverflowMenuItem
                      itemText="Delete"
                      isDelete
                      onClick={() => setPendingAction('delete')}
                    />
                  </OverflowMenu>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        ) : (
          <EmptyState
            icon={Add}
            title="No authentication configured"
            subtitle="Create authentication to verify sessions before initialization."
          />
        )}
      </TableSection>
    </div>
  );
};

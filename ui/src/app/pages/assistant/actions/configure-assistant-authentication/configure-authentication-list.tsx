import { FC, useEffect, useMemo, useState } from 'react';
import { AssistantConfiguration } from '@rapidaai/react';
import {
  Button as CarbonButton,
  ModalBody,
  ModalFooter,
  ModalHeader,
  ComposedModal,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  OverflowMenu,
  OverflowMenuItem,
} from '@carbon/react';
import { Add } from '@carbon/icons-react';
import toast from 'react-hot-toast/headless';

import { useCurrentCredential } from '@/hooks/use-credential';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { UrlTableCell } from '@/app/components/ui/table';
import { SectionLoader } from '@/app/components/ui/feedback';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { EmptyState } from '@/app/components/ui/feedback';
import { RecordStatusIndicator } from '@/app/components/ui/feedback';
import { toHumanReadableDateTime } from '@/utils/date';

import {
  toOptionMap,
  AUTH_OPTION_ENDPOINT,
  AUTH_OPTION_METHOD,
} from './shared';
import {
  deleteAssistantConfigurationById,
  listAssistantConfigurations,
  updateAssistantConfigurationEnabled,
} from '@/clients/assistant.client';
import { AssistantSettingsShellHeader } from '@/app/pages/assistant/components/settings-shell-header';

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
    listAssistantConfigurations({
      assistantId,
      configurationType: authenticationConfigurationType,
      page: 1,
      pageSize: 1,
      criteria: [],
      auth: { projectId, token, userId: authId },
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

    setSubmittingAction(true);
    deleteAssistantConfigurationById({
      assistantId,
      configurationId: authentication.getId(),
      auth: { projectId, token, userId: authId },
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

    setSubmittingAction(true);
    updateAssistantConfigurationEnabled({
      assistantId,
      configurationType: authenticationConfigurationType,
      configuration: authentication,
      enabled,
      auth: { projectId, token, userId: authId },
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
      <AssistantSettingsShellHeader
        title="Authentication"
        href={`/deployment/assistant/${assistantId}/configure-authentication`}
        onReload={load}
        reloadLabel="Reload authentication"
        primaryAction={{
          label: 'Configure authentication',
          onClick: () => navigator.goTo(configureRoute),
        }}
      />

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

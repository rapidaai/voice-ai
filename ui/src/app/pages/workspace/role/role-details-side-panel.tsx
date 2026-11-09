import { useEffect, useRef, useState, type RefObject } from 'react';
import {
  RoleDefinition,
  RoleDefinitionType,
  RoleScope,
  RoleSubject,
  ServiceError,
} from '@rapidaai/react';
import { ApiKey, User } from '@carbon/icons-react';
import { Button, InlineLoading, Tag } from '@carbon/react';
import { OverviewRow, RightSideModal } from '@/app/components/dialogs/shared';
import { Tabs } from '@/app/components/ui/primitives';
import { getAccessRole } from '@/clients';
import { useCredential } from '@/hooks/use-credential';
import { toHumanReadableDateTime } from '@/utils/date';

type PanelStatus =
  | 'idle'
  | 'loading'
  | 'ready'
  | 'error'
  | 'denied'
  | 'not-found';

interface RoleDetailsSidePanelProps {
  open: boolean;
  roleDefinitionId: string | null;
  launcherButtonRef: RefObject<HTMLButtonElement>;
  onClose: () => void;
}

function actorName(
  actor: ReturnType<RoleDefinition['getCreatedactor']>,
  fallback: string,
) {
  if (!actor) return fallback;
  return actor.getDisplayname() || `${actor.getType()} ${actor.getId()}`;
}

function responseStatus(response: {
  getCode?: () => number;
  getError?: () => { getErrorcode?: () => string } | undefined;
}) {
  return Number(
    response.getCode?.() || response.getError?.()?.getErrorcode?.() || 0,
  );
}

const tabsClassName =
  '[&_.cds--tabs__nav]:border-b [&_.cds--tabs__nav]:border-border-subtle [&_.cds--tab-content]:!h-full [&_.cds--tab-content]:!min-h-0 [&_.cds--tab-content]:!p-0';

export function RoleDetailsSidePanel({
  open,
  roleDefinitionId,
  launcherButtonRef,
  onClose,
}: RoleDetailsSidePanelProps) {
  const [userId, token] = useCredential();
  const [role, setRole] = useState<RoleDefinition | null>(null);
  const [status, setStatus] = useState<PanelStatus>('idle');
  const [error, setError] = useState('');
  const [loadCycle, setLoadCycle] = useState(0);
  const [selectedTab, setSelectedTab] = useState(0);
  const requestSequence = useRef(0);

  useEffect(() => {
    if (!open || !roleDefinitionId) {
      requestSequence.current += 1;
      setRole(null);
      setStatus('idle');
      setError('');
      return;
    }

    const requestId = ++requestSequence.current;
    setRole(null);
    setStatus('loading');
    setError('');
    setSelectedTab(0);
    getAccessRole({
      roleDefinitionId,
      auth: { token, userId },
    })
      .then(response => {
        if (requestId !== requestSequence.current) return;
        if (!response.getSuccess() || !response.getRole()) {
          const code = responseStatus(response);
          if (code === 404 || code === 5) {
            setStatus('not-found');
            return;
          }
          if (code === 403 || code === 7) {
            setStatus('denied');
            return;
          }
          setError(
            response.getError()?.getHumanmessage() || 'Unable to load role.',
          );
          setStatus('error');
          return;
        }
        setRole(response.getRole()!);
        setStatus('ready');
      })
      .catch((requestError: ServiceError) => {
        if (requestId !== requestSequence.current) return;
        if (requestError.code === 5) {
          setStatus('not-found');
          return;
        }
        if (requestError.code === 7) {
          setStatus('denied');
          return;
        }
        setError(requestError.message || 'Unable to load role.');
        setStatus('error');
      });

    return () => {
      requestSequence.current += 1;
    };
  }, [loadCycle, open, roleDefinitionId, token, userId]);

  const close = () => {
    requestSequence.current += 1;
    onClose();
    launcherButtonRef.current?.focus();
  };

  const isBuiltIn =
    role?.getDefinitiontype() ===
    RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN;
  const permissions = role?.getPermissionsList() || [];

  return (
    <RightSideModal
      modalOpen={open}
      setModalOpen={nextOpen => {
        if (!nextOpen) close();
      }}
      className="w-[580px] max-w-[calc(100vw-2rem)]"
      label="Role details"
      title={role?.getName() || roleDefinitionId || 'Role'}
    >
      <div className="relative flex min-h-0 flex-1 flex-col">
        {status === 'loading' && (
          <div className="p-4">
            <InlineLoading description="Loading role details" />
          </div>
        )}

        {status === 'not-found' && (
          <div className="p-5">
            <div>
              <h2 className="text-lg font-semibold">Role unavailable</h2>
              <p className="mt-2 text-sm text-muted">
                This role is no longer available. It may have been archived or
                removed from your organization.
              </p>
            </div>
          </div>
        )}

        {status === 'denied' && (
          <div className="p-5">
            <p role="alert" className="text-sm text-support-error">
              You do not have permission to view this role.
            </p>
          </div>
        )}

        {status === 'error' && (
          <div className="flex flex-col gap-4 p-5">
            <p role="alert" className="text-sm text-support-error">
              {error}
            </p>
            <Button
              kind="primary"
              type="button"
              onClick={() => setLoadCycle(value => value + 1)}
            >
              Retry
            </Button>
          </div>
        )}

        {status === 'ready' && role && (
          <Tabs
            tabs={['Overview', 'Permissions']}
            selectedIndex={selectedTab}
            onChange={setSelectedTab}
            contained
            fill
            aria-label="Role details tabs"
            className={tabsClassName}
            panelClassName="!h-full !min-h-0 !overflow-auto !p-0"
          >
            <div className="w-full divide-y divide-border-subtle">
              <OverviewRow
                label="Description"
                className="h-auto min-h-12 items-start py-3"
              >
                <span className="max-w-[20rem] text-right text-sm text-foreground">
                  {role.getDescription() || 'No role description.'}
                </span>
              </OverviewRow>
              <OverviewRow label="Key">
                <span className="font-mono text-sm text-foreground">
                  {role.getKey()}
                </span>
              </OverviewRow>
              <OverviewRow label="Type">
                <Tag size="sm" type={isBuiltIn ? 'cool-gray' : 'purple'}>
                  {isBuiltIn ? 'Built-in' : 'Custom'}
                </Tag>
              </OverviewRow>
              <OverviewRow label="Scope">
                <span className="text-sm text-foreground">
                  {role.getScope() === RoleScope.ROLE_SCOPE_ORGANIZATION
                    ? 'Organization'
                    : 'Project'}
                </span>
              </OverviewRow>
              <OverviewRow label="Subject">
                <Tag
                  size="md"
                  type="gray"
                  renderIcon={
                    role.getSubject() === RoleSubject.ROLE_SUBJECT_PROJECT_TOKEN
                      ? ApiKey
                      : User
                  }
                >
                  {role.getSubject() === RoleSubject.ROLE_SUBJECT_PROJECT_TOKEN
                    ? 'Project token'
                    : 'User'}
                </Tag>
              </OverviewRow>
              <OverviewRow label="Created by">
                <span className="text-sm text-foreground">
                  {actorName(role.getCreatedactor(), 'System')}
                </span>
              </OverviewRow>
              <OverviewRow label="Created">
                <span className="text-sm text-foreground">
                  {role.getCreateddate()
                    ? toHumanReadableDateTime(role.getCreateddate()!)
                    : 'Unknown'}
                </span>
              </OverviewRow>
              <OverviewRow label="Updated by">
                <span className="text-sm text-foreground">
                  {actorName(role.getUpdatedactor(), 'Not updated')}
                </span>
              </OverviewRow>
              <OverviewRow label="Updated">
                <span className="text-sm text-foreground">
                  {role.getUpdateddate()
                    ? toHumanReadableDateTime(role.getUpdateddate()!)
                    : 'Never'}
                </span>
              </OverviewRow>
            </div>
            <div className="w-full divide-y divide-border-subtle">
              {permissions.length === 0 ? (
                <p className="p-4 text-sm text-muted">
                  No permissions assigned.
                </p>
              ) : (
                <ul className="divide-y divide-border-subtle">
                  {permissions.map(permission => (
                    <li key={permission.getId()} className="px-4 py-4">
                      <p className="text-sm font-medium">
                        {permission.getName()}
                      </p>
                      <p className="mt-1 font-mono text-xs text-muted">
                        {permission.getKey()}
                      </p>
                      <p className="mt-2 text-sm text-muted">
                        {permission.getDescription() ||
                          'No permission description.'}
                      </p>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </Tabs>
        )}
      </div>
    </RightSideModal>
  );
}

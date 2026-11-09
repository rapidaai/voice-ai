import { useEffect, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  RoleDefinition,
  RoleDefinitionType,
  RoleScope,
  RoleSubject,
  ServiceError,
} from '@rapidaai/react';
import {
  Add,
  ApiKey,
  Edit,
  Renew,
  Security,
  TrashCan,
  User,
  View,
} from '@carbon/icons-react';
import {
  Button,
  DataTable,
  DataTableSkeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableToolbar,
  TableToolbarContent,
  Tag,
} from '@carbon/react';
import { Helmet } from '@/app/components/app-shell/helmet';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import {
  ScrollableTableSection,
  TableSection,
} from '@/app/components/layout/sections/table-section';
import { EmptyState } from '@/app/components/ui/feedback';
import {
  IconOnlyButton,
  Pagination,
  Text,
} from '@/app/components/ui/primitives';
import { listAccessRoles } from '@/clients';
import { useCredential } from '@/hooks/use-credential';
import { toHumanReadableDateTime } from '@/utils/date';
import { CreateRoleModal } from './create';
import { DeleteRoleDialog, EditRoleModal } from './detail';
import { RoleDetailsSidePanel } from './role-details-side-panel';
import { RoleQuerySearch, type RoleSearchCriteria } from './role-query-search';

const headers = [
  { key: 'name', header: 'Name' },
  { key: 'scope', header: 'Scope' },
  { key: 'subject', header: 'Subject' },
  { key: 'type', header: 'Type' },
  { key: 'actions', header: 'Actions' },
  { key: 'created', header: 'Created' },
  { key: 'updated', header: 'Updated' },
];

const roleColumnClassName: Record<string, string> = {
  name: 'min-w-56 whitespace-nowrap',
  scope: 'min-w-36 whitespace-nowrap',
  subject: 'min-w-40 whitespace-nowrap',
  type: 'min-w-28 whitespace-nowrap',
  actions: 'w-36 min-w-36 whitespace-nowrap',
  created: 'min-w-36 whitespace-nowrap',
  updated: 'min-w-36 whitespace-nowrap',
};

type RoleLocationState = {
  createdRoleDefinitionId?: string;
};

export function RoleCatalogPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [userId, token] = useCredential();
  const [roles, setRoles] = useState<RoleDefinition[]>([]);
  const [querySearchValue, setQuerySearchValue] = useState('');
  const [criteria, setCriteria] = useState<RoleSearchCriteria[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [totalItems, setTotalItems] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [hasCompletedInitialLoad, setHasCompletedInitialLoad] = useState(false);
  const [error, setError] = useState('');
  const [refresh, setRefresh] = useState(0);
  const [selectedRoleDefinitionId, setSelectedRoleDefinitionId] = useState<
    string | null
  >(null);
  const [editingRole, setEditingRole] = useState<RoleDefinition | null>(null);
  const [deletingRole, setDeletingRole] = useState<RoleDefinition | null>(null);
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const requestSequence = useRef(0);
  const launcherButtonRef = useRef<HTMLButtonElement | null>(null);
  const createLauncherButtonRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    const createdRoleDefinitionId = (location.state as RoleLocationState | null)
      ?.createdRoleDefinitionId;
    if (!createdRoleDefinitionId) return;

    setIsCreateOpen(false);
    launcherButtonRef.current = null;
    setSelectedRoleDefinitionId(createdRoleDefinitionId);
    setRefresh(value => value + 1);
    navigate('/organization/roles', { replace: true, state: null });
  }, [location.state, navigate]);

  useEffect(() => {
    const requestId = ++requestSequence.current;
    setIsLoading(true);
    setError('');
    listAccessRoles({
      page,
      pageSize,
      criteria,
      auth: { token, userId },
    })
      .then(response => {
        if (requestId !== requestSequence.current) return;
        setIsLoading(false);
        setHasCompletedInitialLoad(true);
        if (!response.getSuccess()) {
          setError(
            response.getError()?.getHumanmessage() || 'Unable to load roles.',
          );
          return;
        }
        setRoles(response.getRolesList());
        setTotalItems(response.getPaginated()?.getTotalitem() || 0);
      })
      .catch((requestError: ServiceError) => {
        if (requestId !== requestSequence.current) return;
        setIsLoading(false);
        setHasCompletedInitialLoad(true);
        setError(
          requestError.code === 7
            ? 'You do not have permission to manage roles.'
            : requestError.message || 'Unable to load roles.',
        );
      });
    return () => {
      requestSequence.current += 1;
    };
  }, [criteria, page, pageSize, refresh, token, userId]);

  const rows = roles.map(role => ({
    id: role.getId(),
    name: role.getName(),
    scope:
      role.getScope() === RoleScope.ROLE_SCOPE_ORGANIZATION
        ? 'Organization'
        : 'Project',
    subject:
      role.getSubject() === RoleSubject.ROLE_SUBJECT_PROJECT_TOKEN
        ? 'Project token'
        : 'User',
    type:
      role.getDefinitiontype() ===
      RoleDefinitionType.ROLE_DEFINITION_TYPE_BUILT_IN
        ? 'Built-in'
        : 'Custom',
    actions: '',
    created: role.getCreateddate()
      ? toHumanReadableDateTime(role.getCreateddate()!)
      : '-',
    updated: role.getUpdateddate()
      ? toHumanReadableDateTime(role.getUpdateddate()!)
      : '-',
  }));

  const createRole = () => setIsCreateOpen(true);

  const toolbar = (
    <PageHeaderStickyToolbar>
      <TableToolbar>
        <TableToolbarContent>
          <RoleQuerySearch
            value={querySearchValue}
            onChange={setQuerySearchValue}
            onApply={filters => {
              setPage(1);
              setCriteria(filters);
            }}
          />
          <Button
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh roles"
            kind="ghost"
            tooltipPosition="bottom"
            onClick={() => setRefresh(value => value + 1)}
          />
        </TableToolbarContent>
      </TableToolbar>
    </PageHeaderStickyToolbar>
  );

  return (
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden bg-surface"
    >
      <Helmet title="Roles & Permissions" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Organization
          </Text>
        }
        title="Roles & Permissions"
        description="Define reusable roles from the permissions available in your organization and projects."
        contextualActions={
          <Text variant="label-01" className="text-muted tabular-nums">
            {roles.length} of {totalItems}
          </Text>
        }
        pageActions={
          <Button
            ref={createLauncherButtonRef}
            className="w-full md:w-auto"
            size="md"
            renderIcon={Add}
            onClick={createRole}
          >
            Create role
          </Button>
        }
      />

      {isLoading && !hasCompletedInitialLoad ? (
        <TableSection className="min-h-0 overflow-hidden bg-surface">
          <DataTableSkeleton
            headers={headers}
            columnCount={headers.length}
            rowCount={5}
            showHeader={false}
            showToolbar
            size="lg"
            className="w-full"
          />
        </TableSection>
      ) : (
        <TableSection className="min-h-0 bg-surface">
          {toolbar}
          {error ? (
            <EmptyState
              icon={Security}
              title={error}
              action="Try again"
              actionIcon={Renew}
              onAction={() => setRefresh(value => value + 1)}
            />
          ) : rows.length === 0 ? (
            <EmptyState
              icon={Security}
              title="No roles found"
              subtitle="Change the filters or create a custom role."
              action="Create role"
              actionIcon={Add}
              onAction={createRole}
            />
          ) : (
            <>
              <ScrollableTableSection className="shrink-0">
                <DataTable rows={rows} headers={headers}>
                  {({
                    rows: tableRows,
                    headers: tableHeaders,
                    getTableProps,
                    getHeaderProps,
                    getRowProps,
                  }) => (
                    <Table {...getTableProps()} className="min-w-max">
                      <TableHead>
                        <TableRow>
                          {tableHeaders.map(header => (
                            <TableHeader
                              {...getHeaderProps({ header })}
                              key={header.key}
                              className={roleColumnClassName[header.key]}
                            >
                              {header.header}
                            </TableHeader>
                          ))}
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {tableRows.map(row => {
                          const currentRole = roles.find(
                            role => role.getId() === row.id,
                          );
                          if (!currentRole) return null;
                          const isCustomRole =
                            currentRole.getDefinitiontype() ===
                            RoleDefinitionType.ROLE_DEFINITION_TYPE_CUSTOM;

                          return (
                            <TableRow {...getRowProps({ row })} key={row.id}>
                              {row.cells.map(cell => (
                                <TableCell
                                  key={cell.id}
                                  className={
                                    cell.info.header === 'created' ||
                                    cell.info.header === 'updated'
                                      ? 'min-w-36 whitespace-nowrap text-[13px]'
                                      : 'text-sm'
                                  }
                                >
                                  {cell.info.header === 'actions' ? (
                                    <div className="flex items-center gap-0">
                                      <IconOnlyButton
                                        size="md"
                                        kind="ghost"
                                        renderIcon={View}
                                        iconDescription={`View ${currentRole.getName()}`}
                                        tooltipPosition="bottom"
                                        onClick={event => {
                                          launcherButtonRef.current =
                                            event.currentTarget;
                                          setSelectedRoleDefinitionId(
                                            currentRole.getId(),
                                          );
                                        }}
                                      />
                                      <IconOnlyButton
                                        size="md"
                                        kind="ghost"
                                        renderIcon={Edit}
                                        iconDescription={`Edit ${currentRole.getName()}`}
                                        tooltipPosition="bottom"
                                        disabled={!isCustomRole}
                                        onClick={() =>
                                          setEditingRole(currentRole)
                                        }
                                      />
                                      <IconOnlyButton
                                        size="md"
                                        kind="ghost"
                                        renderIcon={TrashCan}
                                        iconDescription={`Delete ${currentRole.getName()}`}
                                        tooltipPosition="bottom"
                                        disabled={!isCustomRole}
                                        onClick={() =>
                                          setDeletingRole(currentRole)
                                        }
                                      />
                                    </div>
                                  ) : cell.info.header === 'subject' ? (
                                    <Tag
                                      size="md"
                                      type="gray"
                                      renderIcon={
                                        cell.value === 'Project token'
                                          ? ApiKey
                                          : User
                                      }
                                    >
                                      {cell.value}
                                    </Tag>
                                  ) : cell.info.header === 'type' ? (
                                    <Tag
                                      size="sm"
                                      type={
                                        cell.value === 'Built-in'
                                          ? 'cool-gray'
                                          : 'purple'
                                      }
                                    >
                                      {cell.value}
                                    </Tag>
                                  ) : (
                                    cell.value
                                  )}
                                </TableCell>
                              ))}
                            </TableRow>
                          );
                        })}
                      </TableBody>
                    </Table>
                  )}
                </DataTable>
              </ScrollableTableSection>
              <Pagination
                sticky
                page={page}
                pageSize={pageSize}
                pageSizes={[10, 25, 50, 100]}
                totalItems={totalItems}
                onChange={value => {
                  setPage(value.page);
                  setPageSize(value.pageSize);
                }}
              />
            </>
          )}
        </TableSection>
      )}

      <RoleDetailsSidePanel
        open={Boolean(selectedRoleDefinitionId)}
        roleDefinitionId={selectedRoleDefinitionId}
        launcherButtonRef={launcherButtonRef}
        onClose={() => setSelectedRoleDefinitionId(null)}
      />
      <CreateRoleModal
        launcherButtonRef={createLauncherButtonRef}
        open={isCreateOpen}
        onClose={() => setIsCreateOpen(false)}
      />
      {editingRole && (
        <EditRoleModal
          open
          role={editingRole}
          onClose={() => setEditingRole(null)}
          onUpdated={() => setRefresh(value => value + 1)}
        />
      )}
      {deletingRole && (
        <DeleteRoleDialog
          open
          role={deletingRole}
          onClose={() => setDeletingRole(null)}
          onDeleted={() => setRefresh(value => value + 1)}
        />
      )}
    </div>
  );
}

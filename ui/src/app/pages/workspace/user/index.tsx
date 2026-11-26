import { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { InviteOrganizationUserDialog } from '@/app/components/dialogs/workspace';
import { InviteProjectUserDialog } from '@/app/components/dialogs/workspace';
import { ManageOrganizationUserRolesDialog } from '@/app/components/dialogs/workspace';
import { User } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import { useUserPageStore } from '@/stores/user';
import { SingleUser } from '@/app/pages/workspace/user/single-user';
import { Pagination, Text } from '@/app/components/ui/primitives';
import { Add, Edit, Renew, TrashCan, UserMultiple } from '@carbon/icons-react';
import {
  Table,
  TableHead,
  TableRow,
  TableHeader,
  TableBody,
  TableToolbar,
  TableBatchAction,
  TableBatchActions,
  TableToolbarContent,
  Button,
  DataTable,
  DataTableSkeleton,
  TableSelectRow,
} from '@carbon/react';
import {
  PageHeader,
  PageHeaderStickyToolbar,
} from '@/app/components/layout/page-header';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { ConfirmDeleteDialog } from '@/app/components/dialogs/shared';
import { deleteOrganizationUser } from '@/clients';
import { EmptyState } from '@/app/components/ui/feedback';
import {
  UserQuerySearch,
  type OrganizationUserSearchFilter,
} from './user-query-search';

const headers = [
  { key: 'id', header: 'ID' },
  { key: 'name', header: 'Name' },
  { key: 'email', header: 'Email' },
  { key: 'role', header: 'Organization Roles' },
  { key: 'createdDate', header: 'Date Created' },
  { key: 'status', header: 'Status' },
];

const skeletonHeaders = [{ key: 'select', header: '' }, ...headers];

const fallbackErrorMessage =
  'Unable to process your request. please try again later.';

export function UserPage() {
  const { showLoader, hideLoader } = useRapidaStore();
  const [inviteOrganizationModalOpen, setInviteOrganizationModalOpen] =
    useState(false);
  const [userPendingProjectInvite, setUserPendingProjectInvite] =
    useState<User | null>(null);
  const [userPendingRoleUpdate, setUserPendingRoleUpdate] =
    useState<User | null>(null);
  const [userPendingDelete, setUserPendingDelete] = useState<User | null>(null);
  const [isFetching, setIsFetching] = useState(true);
  const [querySearchValue, setQuerySearchValue] = useState('');
  const [searchFilters, setSearchFilters] = useState<
    OrganizationUserSearchFilter[]
  >([]);
  const { organizationId, projectId, authId, token } = useCurrentCredential();
  const userActions = useUserPageStore();

  const onError = useCallback((err: string) => {
    setIsFetching(false);
    toast.error(err);
  }, []);

  const onSuccess = useCallback(() => {
    setIsFetching(false);
  }, []);

  const getUsers = useCallback(() => {
    setIsFetching(true);
    userActions.getAllUser(
      token,
      authId,
      organizationId,
      projectId,
      onError,
      onSuccess,
    );
  }, [
    authId,
    onError,
    onSuccess,
    organizationId,
    projectId,
    token,
    userActions.getAllUser,
  ]);

  useEffect(() => {
    getUsers();
  }, [getUsers, userActions.page, userActions.pageSize, userActions.criteria]);

  const onDeleteOrganizationUser = async (user: User) => {
    showLoader('overlay');

    try {
      const response = await deleteOrganizationUser({
        userId: user.getId(),
        auth: { organizationId, token, userId: authId },
      });
      hideLoader();

      const responseError = response.getError();
      const message = responseError?.getHumanmessage() || fallbackErrorMessage;

      if (response.getSuccess()) {
        setUserPendingDelete(null);
        toast.success('The user was deleted from the organization.');
        getUsers();
        return;
      }

      toast.error(message);
    } catch (err: any) {
      hideLoader();
      toast.error(err?.message || fallbackErrorMessage);
    }
  };

  const tableRows = userActions.users.map(user => ({
    id: user.getId(),
    name: user.getName(),
    email: user.getEmail(),
    role: user
      .getOrganizationrolesList()
      .map(role => role.getName())
      .join(', '),
    createdDate: user.getCreateddate()?.toString() || '',
    status: user.getStatus?.() || '',
  }));
  const visibleTableRows = tableRows.filter(row =>
    searchFilters.every(filter => {
      const currentValue = String(row[filter.key] || '').toLowerCase();
      const expectedValue = filter.value.toLowerCase();
      return filter.logic === 'contains'
        ? currentValue.includes(expectedValue)
        : currentValue === expectedValue;
    }),
  );

  return (
    <div
      data-page-scroll
      className="flex min-h-0 flex-1 flex-col overflow-y-auto overflow-x-hidden bg-surface"
    >
      <Helmet title="User and Teams" />
      <PageHeader
        eyebrow={
          <Text variant="label-01" className="text-primary">
            Organization
          </Text>
        }
        title="Users"
        description="Manage organization access, roles, and project collaboration for your team."
        contextualActions={
          <Text variant="label-01" className="text-muted tabular-nums">
            {visibleTableRows.length} of {userActions.totalCount}
          </Text>
        }
        pageActions={
          <Button
            className="w-full md:w-auto"
            size="md"
            renderIcon={Add}
            onClick={() => setInviteOrganizationModalOpen(true)}
          >
            Invite user
          </Button>
        }
      />
      {isFetching ? (
        <TableSection className="min-h-0 overflow-hidden bg-surface">
          <DataTableSkeleton
            headers={skeletonHeaders}
            columnCount={skeletonHeaders.length}
            rowCount={5}
            showHeader={false}
            showToolbar
            size="lg"
            className="w-full"
          />
        </TableSection>
      ) : (
        <TableSection className="min-h-0 bg-surface">
          <DataTable rows={visibleTableRows} headers={headers} radio>
            {({
              rows,
              headers: tableHeaders,
              selectedRows,
              getBatchActionProps,
              getHeaderProps,
              getRowProps,
              getSelectionProps,
              getTableProps,
              getToolbarProps,
            }) => {
              const batchActionProps = getBatchActionProps();
              const selectedUser = selectedRows[0]
                ? userActions.users.find(
                    user => user.getId() === selectedRows[0].id,
                  )
                : undefined;
              const selectedUserIsOwner = selectedUser
                ?.getOrganizationrolesList()
                .some(role => role.getRoledefinitionid() === '101');
              return (
                <>
                  <PageHeaderStickyToolbar>
                    <TableToolbar {...getToolbarProps()}>
                      {selectedUser && (
                        <TableBatchActions
                          {...batchActionProps}
                          shouldShowBatchActions
                        >
                          <TableBatchAction
                            renderIcon={Add}
                            iconDescription="Invite to project"
                            onClick={() => {
                              setUserPendingProjectInvite(selectedUser);
                              batchActionProps.onCancel();
                            }}
                          >
                            Invite to project
                          </TableBatchAction>
                          {!selectedUserIsOwner && (
                            <TableBatchAction
                              renderIcon={Edit}
                              iconDescription="Manage roles"
                              onClick={() => {
                                setUserPendingRoleUpdate(selectedUser);
                                batchActionProps.onCancel();
                              }}
                            >
                              Manage roles
                            </TableBatchAction>
                          )}
                          {!selectedUserIsOwner && (
                            <TableBatchAction
                              renderIcon={TrashCan}
                              iconDescription="Delete"
                              onClick={() => {
                                setUserPendingDelete(selectedUser);
                                batchActionProps.onCancel();
                              }}
                            >
                              Delete
                            </TableBatchAction>
                          )}
                        </TableBatchActions>
                      )}
                      <TableToolbarContent>
                        <UserQuerySearch
                          value={querySearchValue}
                          onChange={setQuerySearchValue}
                          onApply={filters => {
                            userActions.setPage(1);
                            setSearchFilters(filters);
                          }}
                        />
                        <Button
                          hasIconOnly
                          renderIcon={Renew}
                          iconDescription="Refresh"
                          kind="ghost"
                          onClick={getUsers}
                          tooltipPosition="bottom"
                        />
                      </TableToolbarContent>
                    </TableToolbar>
                  </PageHeaderStickyToolbar>

                  {visibleTableRows.length > 0 ? (
                    <>
                      <div className="shrink-0 overflow-x-auto">
                        <Table {...getTableProps()} className="min-w-max">
                          <TableHead>
                            <TableRow>
                              <TableHeader className="!w-12" />
                              {tableHeaders.map(header => (
                                <TableHeader
                                  {...getHeaderProps({ header })}
                                  key={header.key}
                                >
                                  {header.header}
                                </TableHeader>
                              ))}
                            </TableRow>
                          </TableHead>
                          <TableBody>
                            {rows.map(tableRow => {
                              const user = userActions.users.find(
                                candidate => candidate.getId() === tableRow.id,
                              );
                              if (!user) return null;

                              return (
                                <TableRow
                                  {...getRowProps({ row: tableRow })}
                                  key={tableRow.id}
                                >
                                  <TableSelectRow
                                    {...getSelectionProps({ row: tableRow })}
                                  />
                                  <SingleUser user={user} />
                                </TableRow>
                              );
                            })}
                          </TableBody>
                        </Table>
                      </div>
                      <Pagination
                        sticky
                        totalItems={userActions.totalCount}
                        page={userActions.page}
                        pageSize={userActions.pageSize}
                        pageSizes={[10, 20, 50]}
                        onChange={({ page, pageSize }) => {
                          if (pageSize !== userActions.pageSize) {
                            userActions.setPageSize(pageSize);
                          } else {
                            userActions.setPage(page);
                          }
                        }}
                      />
                    </>
                  ) : (
                    <div className="flex flex-1 items-center justify-center">
                      <EmptyState
                        icon={UserMultiple}
                        title={
                          searchFilters.length > 0
                            ? 'No users found'
                            : 'No organization users'
                        }
                        subtitle={
                          searchFilters.length > 0
                            ? 'No users match your current filters.'
                            : 'Invite teammates to collaborate on projects and manage organization access.'
                        }
                        action="Invite user"
                        actionIcon={Add}
                        onAction={() => setInviteOrganizationModalOpen(true)}
                      />
                    </div>
                  )}
                </>
              );
            }}
          </DataTable>
        </TableSection>
      )}
      <InviteOrganizationUserDialog
        modalOpen={inviteOrganizationModalOpen}
        setModalOpen={setInviteOrganizationModalOpen}
        onSuccess={() => {
          getUsers();
        }}
      />
      <InviteProjectUserDialog
        modalOpen={Boolean(userPendingProjectInvite)}
        setModalOpen={open => {
          if (!open) {
            setUserPendingProjectInvite(null);
          }
        }}
        user={userPendingProjectInvite}
        onSuccess={() => {
          setUserPendingProjectInvite(null);
          getUsers();
        }}
      />
      <ManageOrganizationUserRolesDialog
        modalOpen={Boolean(userPendingRoleUpdate)}
        setModalOpen={open => {
          if (!open) {
            setUserPendingRoleUpdate(null);
          }
        }}
        user={userPendingRoleUpdate}
        onSuccess={() => {
          setUserPendingRoleUpdate(null);
          getUsers();
        }}
      />
      <ConfirmDeleteDialog
        showing={Boolean(userPendingDelete)}
        title="Delete user"
        content={
          userPendingDelete
            ? `This will delete "${userPendingDelete.getEmail()}" from the organization. Type the email to confirm.`
            : ''
        }
        objectName={userPendingDelete?.getEmail() || ''}
        confirmText="Delete user"
        onConfirm={() => {
          if (userPendingDelete) {
            onDeleteOrganizationUser(userPendingDelete);
          }
        }}
        onCancel={() => {
          setUserPendingDelete(null);
        }}
        onClose={() => {
          setUserPendingDelete(null);
        }}
      />
    </div>
  );
}

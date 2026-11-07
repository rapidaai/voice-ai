import { useCallback, useEffect, useState } from 'react';
import { Helmet } from '@/app/components/app-shell/helmet';
import { InviteOrganizationUserDialog } from '@/app/components/dialogs/workspace';
import { InviteProjectUserDialog } from '@/app/components/dialogs/workspace';
import { User } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { useRapidaStore } from '@/stores/app';
import { useUserPageStore } from '@/stores/user';
import { SingleUser } from '@/app/pages/workspace/user/single-user';
import { Pagination } from '@/app/components/ui/primitives';
import {
  Add,
  Renew,
  TrashCan,
  UserAdmin,
  UserMultiple,
} from '@carbon/icons-react';
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
  TableToolbarSearch,
  Button,
  DataTable,
  DataTableSkeleton,
  TableSelectRow,
} from '@carbon/react';
import { PageHeaderBlock } from '@/app/components/layout/blocks/page-header-block';
import { PageTitleWithCount } from '@/app/components/layout/blocks/page-title-with-count';
import { TableSection } from '@/app/components/layout/sections/table-section';
import { ConfirmDeleteDialog } from '@/app/components/dialogs/shared';
import { deleteOrganizationUser, updateOrganizationUserRole } from '@/clients';
import { EmptyState } from '@/app/components/ui/feedback';

const headers = [
  { key: 'id', header: 'ID' },
  { key: 'name', header: 'Name' },
  { key: 'email', header: 'Email' },
  { key: 'role', header: 'Org Role' },
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
  const [userPendingDelete, setUserPendingDelete] = useState<User | null>(null);
  const [isFetching, setIsFetching] = useState(true);
  const { projectId, authId, token } = useCurrentCredential();
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
    userActions.getAllUser(token, authId, projectId, onError, onSuccess);
  }, [authId, onError, onSuccess, projectId, token, userActions.getAllUser]);

  useEffect(() => {
    getUsers();
  }, [getUsers, userActions.page, userActions.pageSize, userActions.criteria]);

  const onDeleteOrganizationUser = async (user: User) => {
    showLoader('overlay');

    try {
      const response = await deleteOrganizationUser({
        userId: user.getId(),
        auth: { token, userId: authId },
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

  const onUpdateOrganizationRole = async (
    user: User,
    organizationRole: string,
  ) => {
    showLoader('overlay');

    try {
      const response = await updateOrganizationUserRole({
        userId: user.getId(),
        organizationRole,
        auth: { token, userId: authId },
      });
      hideLoader();

      const responseError = response.getError();
      const message = responseError?.getHumanmessage() || fallbackErrorMessage;

      if (response.getSuccess()) {
        toast.success(
          organizationRole === 'admin'
            ? 'The user was promoted to admin.'
            : 'The user was changed to member.',
        );
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
    role: user.getRole(),
    createdDate: user.getCreateddate()?.toString() || '',
    status: user.getStatus?.() || '',
  }));

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col bg-surface">
      <Helmet title="User and Teams" />
      <PageHeaderBlock>
        <PageTitleWithCount
          count={userActions.users.length}
          total={userActions.totalCount}
        >
          Users
        </PageTitleWithCount>
      </PageHeaderBlock>
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
          <DataTable rows={tableRows} headers={headers} radio>
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
              onInputChange,
            }) => {
              const batchActionProps = getBatchActionProps();
              const selectedUser = selectedRows[0]
                ? userActions.users.find(
                    user => user.getId() === selectedRows[0].id,
                  )
                : undefined;
              const selectedOrganizationRole = selectedUser
                ?.getRole()
                ?.toLowerCase();
              const nextOrganizationRole =
                selectedOrganizationRole === 'member'
                  ? 'admin'
                  : selectedOrganizationRole === 'admin'
                    ? 'member'
                    : null;

              return (
                <>
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
                        {nextOrganizationRole && (
                          <TableBatchAction
                            renderIcon={UserAdmin}
                            iconDescription={
                              nextOrganizationRole === 'admin'
                                ? 'Make admin'
                                : 'Make member'
                            }
                            onClick={() => {
                              onUpdateOrganizationRole(
                                selectedUser,
                                nextOrganizationRole,
                              );
                              batchActionProps.onCancel();
                            }}
                          >
                            {nextOrganizationRole === 'admin'
                              ? 'Make admin'
                              : 'Make member'}
                          </TableBatchAction>
                        )}
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
                      </TableBatchActions>
                    )}
                    <TableToolbarContent>
                      <TableToolbarSearch
                        placeholder="Search users..."
                        onChange={onInputChange}
                      />
                      <Button
                        hasIconOnly
                        renderIcon={Renew}
                        iconDescription="Refresh"
                        kind="ghost"
                        onClick={getUsers}
                        tooltipPosition="bottom"
                      />
                      <Button
                        size="md"
                        renderIcon={Add}
                        onClick={() => setInviteOrganizationModalOpen(true)}
                      >
                        Invite user
                      </Button>
                    </TableToolbarContent>
                  </TableToolbar>

                  {userActions.users.length > 0 ? (
                    <>
                      <div className="min-h-0 flex-1 overflow-auto">
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
                        className="shrink-0"
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
                        title="No organization users"
                        subtitle="Invite teammates to collaborate on projects and manage organization access."
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

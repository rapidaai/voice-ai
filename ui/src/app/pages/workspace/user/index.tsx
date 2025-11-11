import { useEffect, useState, useCallback } from 'react';
import { Helmet } from '@/app/components/Helmet';
import { InviteUserDialog } from '@/app/components/base/modal/invite-user-modal';
import { User } from '@rapidaai/react';
import { useCurrentCredential } from '@/hooks/use-credential';
import toast from 'react-hot-toast/headless';
import { SearchIconInput } from '@/app/components/Form/Input/IconInput';
import { TablePagination } from '@/app/components/base/tables/table-pagination';
import { useRapidaStore } from '@/hooks';
import { BluredWrapper } from '@/app/components/Wrapper/BluredWrapper';
import { useUserPageStore } from '@/hooks';
import { SingleUser } from '@/app/pages/workspace/user/single-user';
import { IBlueButton, IButton } from '@/app/components/Form/Button';
import { Plus, RotateCw } from 'lucide-react';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { TableSection } from '@/app/components/sections/table-section';
import { Table } from '@/app/components/base/tables/table';
import { TableBody } from '@/app/components/base/tables/table-body';
import { TableHead } from '@/app/components/base/tables/table-head';

/**
 *
 * @returns
 */
export function UserPage() {
  /**
   *loader
   */
  const { showLoader, hideLoader } = useRapidaStore();

  /**
   * for create a user modal
   */
  const [createUserModalOpen, setCreateUserModalOpen] = useState(false);

  /**
   * authentication with token
   */
  const { projectId, authId, token } = useCurrentCredential();
  const userActions = useUserPageStore();
  const onError = useCallback((err: string) => {
    hideLoader();
    toast.error(err);
  }, []);
  const onSuccess = useCallback((data: User[]) => {
    hideLoader();
  }, []);
  /**
   * call the api
   */
  const getUsers = useCallback((token, userId, projectId) => {
    showLoader();
    userActions.getAllUser(token, userId, projectId, onError, onSuccess);
  }, []);

  useEffect(() => {
    getUsers(token, authId, projectId);
  }, [userActions.page, userActions.pageSize, userActions.criteria]);
  /**
   *
   */
  return (
    <>
      <Helmet title="User and Teams" />
      <PageHeaderBlock>
        <div className="flex items-center gap-3">
          <PageTitleBlock>Users</PageTitleBlock>
          <div className="text-xs opacity-75">
            {`${userActions.users.length}/${userActions.totalCount}`}
          </div>
        </div>
        <div className="flex divide-x dark:divide-gray-800">
          <IButton
            onClick={() => {
              getUsers(token, authId, projectId);
            }}
          >
            Reload user
            <RotateCw className="w-4 h-4 ml-1.5" />
          </IButton>
          <IBlueButton
            onClick={() => {
              setCreateUserModalOpen(true);
            }}
          >
            Create new user
            <Plus className="w-4 h-4 ml-1.5" />
          </IBlueButton>
        </div>
      </PageHeaderBlock>
      <BluredWrapper className="p-0">
        <SearchIconInput className="bg-light-background" />
        <TablePagination
          columns={userActions.columns}
          currentPage={userActions.page}
          onChangeCurrentPage={userActions.setPage}
          totalItem={userActions.totalCount}
          pageSize={userActions.pageSize}
          onChangePageSize={userActions.setPageSize}
          onChangeColumns={userActions.setColumns}
        />
      </BluredWrapper>
      <TableSection>
        <Table className="bg-white dark:bg-gray-900">
          <TableHead columns={userActions.columns} isActionable />
          <TableBody>
            {userActions.users.map((usr, idx) => {
              return <SingleUser key={idx} user={usr} />;
            })}
          </TableBody>
        </Table>
      </TableSection>
      <InviteUserDialog
        modalOpen={createUserModalOpen}
        setModalOpen={setCreateUserModalOpen}
        onSuccess={() => {
          getUsers(token, authId, projectId);
        }}
      ></InviteUserDialog>
    </>
  );
}

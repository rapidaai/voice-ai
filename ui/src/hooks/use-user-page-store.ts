import { ConnectionConfig, GetAllUser } from '@rapidaai/react';
import { User } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { GetAllUserResponse } from '@rapidaai/react';

import { UserType } from '@/types';
import { initialPaginated } from '@/types/types.paginated';
import { create } from 'zustand';
import { connectionConfig } from '@/configs';

/**
 *
 */
export const useUserPageStore = create<UserType>((set, get) => ({
  ...initialPaginated,
  /**
   * list of endpoint where these will be part of it
   */
  users: [],

  /**
   *
   * @param number
   * @returns
   */
  setPageSize: (pageSize: number) => {
    // when someone change pagesize change the page to zero
    set({
      page: 1,
      pageSize: pageSize,
    });
  },

  /**
   *
   * @param number
   * @returns
   */
  setPage: (pg: number) => {
    set({
      page: pg,
    });
  },

  /**
   *
   * @param number
   * @returns
   */
  setTotalCount: (tc: number) => {
    set({
      totalCount: tc,
    });
  },

  /**
   *
   * @param k
   * @param v
   */
  addCriteria: (k: string, v: string, logic: string) => {
    let current = get().criteria.filter(x => x.key !== k && x.logic !== logic);
    if (v) current.push({ key: k, value: v, logic: logic });
    set({
      criteria: current,
    });
  },

  /**
   *
   * @param v
   */
  addCriterias: (v: { k: string; v: string; logic: string }[]) => {
    let current = get().criteria.filter(
      x => !v.find(y => y.k === x.key && x.logic === y.logic),
    );
    v.forEach(c => {
      current.push({ key: c.k, value: c.v, logic: c.logic });
    });
    set({
      criteria: current,
    });
  },

  /**
   *
   * @param users
   */
  setUsers: (users: User[]) => {
    set({
      users: users,
    });
  },

  /**
   *
   * @param projectId
   * @param token
   * @param userId
   */
  getAllUser: (
    token: string,
    userId: string,
    projectId: string,
    onError: (err: string) => void,
    onSuccess: (e: User[]) => void,
  ) => {
    const afterGetAllUser = (
      err: ServiceError | null,
      gur: GetAllUserResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        get().setUsers(gur.getDataList());
        let paginated = gur.getPaginated();
        if (paginated) {
          get().setTotalCount(paginated.getTotalitem());
        }
        onSuccess(gur.getDataList());
        return;
      }
      onError(
        'Unable to get all the users for your organization, please try again in sometime.',
      );
    };

    GetAllUser(
      connectionConfig,
      get().page,
      get().pageSize,
      get().criteria,
      afterGetAllUser,
      ConnectionConfig.WithDebugger({
        authorization: token,
        userId: userId,
        projectId: projectId,
      }),
    );
  },

  /**
   * columns
   */
  columns: [
    { name: 'User', key: 'getId', visible: true },
    { name: 'Name', key: 'getName', visible: true },
    { name: 'Email', key: 'getEmail', visible: true },
    { name: 'Role', key: 'getRole', visible: true },
    { name: 'Joined on', key: 'getCreatedDate', visible: true },
    { name: 'Status', key: 'getStatus', visible: true },
  ],

  /**
   *
   * @param cl
   */
  setColumns(cl: { name: string; key: string; visible: boolean }[]) {
    set({
      columns: cl,
    });
  },

  /**
   *
   * @param k
   * @returns
   */
  visibleColumn: (k: string): boolean => {
    const column = get().columns.find(c => c.key === k);
    return column ? column.visible : false;
  },

  /**
   * clear everything from the context
   * @returns
   */
  clear: () => set({}, true),
}));

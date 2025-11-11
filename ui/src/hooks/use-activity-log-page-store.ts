import { create } from 'zustand';
import {
  ActivityLogType,
  ActivityLogTypeProperty,
} from '@/types/types.activity-log';
import { initialPaginated } from '@/types/types.paginated';
import { GetAllAuditLogResponse } from '@rapidaai/react';
import { AuditLog } from '@rapidaai/react';
import { ServiceError } from '@rapidaai/react';
import { ConnectionConfig, GetActivities } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const intialActivityLog: ActivityLogTypeProperty = {
  activities: [],
};

/**
 *
 */
export const useActivityLogPage = create<ActivityLogType>((set, get) => ({
  ...intialActivityLog,
  ...initialPaginated,

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
   * @param ep
   */
  onChangeActivities: (ep: AuditLog[]) => {
    set({
      activities: ep,
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
   * @param projectId
   * @param token
   * @param userId
   */
  getActivities: (
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: AuditLog[]) => void,
  ) => {
    const afterGetAllActivityLog = (
      err: ServiceError | null,
      gur: GetAllAuditLogResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        get().onChangeActivities(gur.getDataList());
        let paginated = gur.getPaginated();
        if (paginated) {
          get().setTotalCount(paginated.getTotalitem());
        }
        onSuccess(gur.getDataList());
      } else {
        let errorMessage = gur?.getError();
        if (errorMessage) {
          onError(errorMessage.getHumanmessage());
          return;
        }
        onError('Unable to get your activity log, please try again later.');
      }
    };

    GetActivities(
      connectionConfig,
      projectId,
      get().page,
      get().pageSize,
      get().criteria,
      afterGetAllActivityLog,
      ConnectionConfig.WithDebugger({
        authorization: token,
        projectId: projectId,
        userId: userId,
      }),
    );
  },

  /**
   * columns
   */
  columns: [
    { name: 'Source', key: 'Source', visible: true },
    { name: 'Provider Name', key: 'Provider Name', visible: true },
    { name: 'Model Name', key: 'Model Name', visible: true },
    { name: 'Created Date', key: 'Created Date', visible: true },
    { name: 'Status', key: 'Status', visible: true },
    { name: 'Time Taken', key: 'Time_Taken', visible: true },
    { name: 'Http status', key: 'Http_status', visible: true },
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
  clear: () => set({ ...intialActivityLog }, true),
}));

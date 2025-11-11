import { create } from 'zustand';
import {} from '@/types/types.activity-log';
import { initialPaginated } from '@/types/types.paginated';
import { ServiceError } from '@rapidaai/react';
import {
  EndpointLogType,
  EndpointLogTypeProperty,
} from '@/types/types.endpoint-log';
import { EndpointLog, GetAllEndpointLogResponse } from '@rapidaai/react';
import { GetAllEndpointLog } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const intialActivityLog: EndpointLogTypeProperty = {
  endpointLogs: [],
};

/**
 *
 */
export const useEndpointLogPage = create<EndpointLogType>((set, get) => ({
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
  onChangeLogs: (lgs: EndpointLog[]) => {
    set({
      endpointLogs: lgs,
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
  getLogs: (
    endpointId: string,
    projectId: string,
    token: string,
    userId: string,
    onError: (err: string) => void,
    onSuccess: (e: EndpointLog[]) => void,
  ) => {
    const afterGetAllActivityLog = (
      err: ServiceError | null,
      gur: GetAllEndpointLogResponse | null,
    ) => {
      if (gur?.getSuccess()) {
        get().onChangeLogs(gur.getDataList());
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
        onError(
          'Unable to get endpoint activity logs, please try again later.',
        );
      }
    };

    GetAllEndpointLog(
      connectionConfig,
      endpointId,
      get().page,
      get().pageSize,
      get().criteria,
      afterGetAllActivityLog,
      {
        authorization: token,
        'x-project-id': projectId,
        'x-auth-id': userId,
      },
    );
  },

  columns: [
    { name: 'ID', key: 'id', visible: true },
    { name: 'Version', key: 'version', visible: true },
    { name: 'Source', key: 'source', visible: true },
    { name: 'Status', key: 'status', visible: true },
    { name: 'Total Time Taken', key: 'timetaken', visible: true },
    { name: 'LLM Total Token', key: 'total_token', visible: true },
    { name: 'LLM Time Taken', key: 'time_taken', visible: true },
    { name: 'Created Date', key: 'created_date', visible: true },
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

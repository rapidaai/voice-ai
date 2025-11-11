import { create } from 'zustand';
import {} from '@/types/types.activity-log';
import { initialPaginated } from '@/types/types.paginated';
import { ServiceError } from '@rapidaai/react';
import {
  WebhookLogType,
  WebhookLogTypeProperty,
} from '@/types/types.webhook-log';
import {
  AssistantWebhookLog,
  GetAllAssistantWebhookLogResponse,
} from '@rapidaai/react';
import { GetAllWebhookLog } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const intialActivityLog: WebhookLogTypeProperty = {
  webhookLogs: [],
};

/**
 *
 */
export const useWebhookLogPage = create<WebhookLogType>((set, get) => ({
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
  onChangeActivities: (lgs: AssistantWebhookLog[]) => {
    set({
      webhookLogs: lgs,
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
    onSuccess: (e: AssistantWebhookLog[]) => void,
  ) => {
    const afterGetAllActivityLog = (
      err: ServiceError | null,
      gur: GetAllAssistantWebhookLogResponse | null,
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
        onError('Unable to get webhook activity logs, please try again later.');
      }
    };

    GetAllWebhookLog(
      connectionConfig,
      projectId,
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

  /**
   *   id: string,
       webhookid: string,
       request?: google_protobuf_struct_pb.Struct.AsObject,
       response?: google_protobuf_struct_pb.Struct.AsObject,
       status: string,
       createddate?: google_protobuf_timestamp_pb.Timestamp.AsObject,
       updateddate?: google_protobuf_timestamp_pb.Timestamp.AsObject,
       assistantid: string,
       projectid: string,
       organizationid: string,
       conversationid: string,
       assetprefix: string,
       event: string,
       responsestatus: string,
       timetaken: string,
       retrycount: number,
   * columns
   */
  columns: [
    { name: 'WebhookID', key: 'webhookid', visible: true },
    { name: 'Session ID', key: 'sessionid', visible: true },
    { name: 'Event', key: 'event', visible: true },
    { name: 'Endpoint', key: 'endpoint', visible: true },
    { name: 'Http status', key: 'responsestatus', visible: true },
    { name: 'Time Taken', key: 'timetaken', visible: true },
    { name: 'Retry Count', key: 'retrycount', visible: true },
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

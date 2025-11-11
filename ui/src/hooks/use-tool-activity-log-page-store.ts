import { create } from 'zustand';
import {
  ToolActivityLogType,
  ToolActivityLogTypeProperty,
} from '@/types/types.tool-activity-log';
import { initialPaginated } from '@/types/types.paginated';
import {
  AssistantToolLog,
  Criteria,
  GetAllAssistantToolLog,
  GetAllAssistantToolLogRequest,
  Paginate,
} from '@rapidaai/react';
import { ConnectionConfig } from '@rapidaai/react';
import { connectionConfig } from '@/configs';

const intialToolActivityLog: ToolActivityLogTypeProperty = {
  activities: [],
};

/**
 *
 */
export const useToolActivityLogPage = create<ToolActivityLogType>(
  (set, get) => ({
    ...intialToolActivityLog,
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
    onChangeActivities: (ep: AssistantToolLog[]) => {
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
      let current = get().criteria.filter(
        x => x.key !== k && x.logic !== logic,
      );
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
      onSuccess: (e: AssistantToolLog[]) => void,
    ) => {
      const req = new GetAllAssistantToolLogRequest();
      req.setProjectid(projectId);

      const paginate = new Paginate();
      get().criteria.forEach(({ key, value, logic }) => {
        const ctr = new Criteria();
        ctr.setKey(key);
        ctr.setValue(value);
        ctr.setLogic(logic);
        req.addCriterias(ctr);
      });

      paginate.setPage(get().page);
      paginate.setPagesize(get().pageSize);
      req.setPaginate(paginate);
      GetAllAssistantToolLog(
        connectionConfig,
        req,
        ConnectionConfig.WithDebugger({
          authorization: token,
          projectId: projectId,
          userId: userId,
        }),
      )
        .then(gur => {
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
        })
        .catch(x => {
          onError('Unable to get your activity log, please try again later.');
        });
    },

    /**
     * columns
     */

    columns: [
      { name: 'Assistant', key: 'assistant_id', visible: true },
      {
        name: 'Session',
        key: 'assistant_conversation_id',
        visible: true,
      },
      { name: 'Tool Name', key: 'assistant_tool_name', visible: true },
      { name: 'Action', key: 'execution_method', visible: true },
      { name: 'Status', key: 'status', visible: true },
      { name: 'Time Taken', key: 'time_taken', visible: true },
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
    clear: () => set({ ...intialToolActivityLog }, true),
  }),
);

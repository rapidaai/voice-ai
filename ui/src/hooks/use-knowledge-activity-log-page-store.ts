import { create } from 'zustand';
import {} from '@/types/types.activity-log';
import { initialPaginated } from '@/types/types.paginated';
import {
  Criteria,
  GetAllKnowledgeLog,
  GetAllKnowledgeLogRequest,
  KnowledgeLog,
  Paginate,
} from '@rapidaai/react';
import { ConnectionConfig } from '@rapidaai/react';
import { connectionConfig } from '@/configs';
import { KnowledgeActivityLogTypeProperty } from '@/types/types.knowledge-activity-log';
import { KnowledgeActivityLogType } from '../types/types.knowledge-activity-log';

const intialActivityLog: KnowledgeActivityLogTypeProperty = {
  activities: [],
};

/**
 *
 */
export const useKnowledgeActivityLogPage = create<KnowledgeActivityLogType>(
  (set, get) => ({
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
    onChangeActivities: (ep: KnowledgeLog[]) => {
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
      onSuccess: (e: KnowledgeLog[]) => void,
    ) => {
      const req = new GetAllKnowledgeLogRequest();
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

      GetAllKnowledgeLog(
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

    //     id BIGINT PRIMARY KEY,
    //     created_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    //     updated_date TIMESTAMP WITH TIME ZONE,
    //     status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    //     created_by BIGINT,
    //     updated_by BIGINT,
    //     project_id BIGINT NOT NULL,
    //     organization_id BIGINT NOT NULL,
    //     knowledge_id BIGINT NOT NULL,
    //     retrieval_method VARCHAR(50),
    //     top_k INTEGER,
    //     score_threshold REAL,
    //     document_count INTEGER,
    //     asset_prefix VARCHAR(200) NOT NULL,
    //     time_taken BIGINT,
    //     additional_data TEXT

    columns: [
      { name: 'ID', key: 'id', visible: false },
      { name: 'Knowledge ID', key: 'knowledge_id', visible: true },
      { name: 'Retrieval Method', key: 'retrieval_method', visible: true },
      { name: 'Top K', key: 'top_k', visible: true },
      { name: 'Score Threshold', key: 'score_threshold', visible: true },
      {
        name: 'Retrieval Document Count',
        key: 'document_count',
        visible: true,
      },
      { name: 'Time Taken', key: 'time_taken', visible: true },
      { name: 'Status', key: 'status', visible: true },
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
  }),
);
